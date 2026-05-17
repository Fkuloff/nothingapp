# Push auto-dismiss: design + behaviour

When a user reads / opens / deletes their way out of "has unread messages in
chat X", the tray notification on their other devices should disappear on its
own. Same for chat-level destructive ops (clear_chat, delete_chat) and for
mark_read.

This document explains the mechanism + known limits, so future work doesn't
re-litigate the trade-offs.

## High-level flow

```
device A (any source)                      backend                            device B (any other)

mark_read / delete_message / clear_chat   ───► WS / REST handler              tray entry for chat X
   ↓                                              │                                    │
local dismissChatNotifications(X)                 │                                    │
   removes current device's tray entry            │                                    │
                                                  ▼                                    │
                                       SendDismiss(userID, X)                          │
                                            (web push + FCM data-only)                 │
                                                  │                                    │
                                                  └──────────────────────────────────► dismiss-push received
                                                                                       │
                                                                                       ├─ web: sw.js getNotifications({tag:'chat-X'}) → close()
                                                                                       └─ native: PushNotifications.removeDeliveredNotifications({tag})
                                                                                                  │
                                                                                                  ▼
                                                                                          tray entry gone
```

## Triggers (backend)

| WS / REST action                | Where                            | Condition                          | Dispatch     |
|--------------------------------|----------------------------------|------------------------------------|--------------|
| `mark_read`                    | `handleMarkRead`                 | always (op wipes ALL unread)       | for self     |
| `delete_message`               | `handleDeleteMessage`            | per recipient where `unread → 0`   | per-recipient fanout |
| `clear_chat` / `delete_chat`   | `chatAction` → `onUnreadDrained` | always (op wipes all participants) | every participant |
| `chat_opened` *(reserved)*     | `handleChatOpened`               | always                             | for self — currently unused by frontend (mark_read already covers it) |

All triggers call `webSocketHandler.fireDismissPush(userID, chatID)`, which spawns
a background goroutine and calls `pushService.SendDismiss(...)`. SendDismiss
fans out to both Web Push (VAPID) and FCM in parallel, with data-only payload
`{type:"dismiss", chat_id, tag:"chat-<id>"}`.

## Tag convention

Every outgoing push (regular notification + dismiss) is tagged
`chat-<chatID>`. Collapsing happens automatically:

- N messages from chat X = 1 tray entry, content updates as new ones arrive
- 1 dismiss for chat X = that 1 entry goes away

Matches Telegram / WhatsApp UX. Per-message dismiss is not supported — by
design.

## Receivers

### Web (Service Worker)

`frontend/public/sw.js` `push` handler branches on `payload.type === 'dismiss'`:

```js
if (payload.type === 'dismiss') {
  const tag = payload.tag || `chat-${payload.chat_id}`
  const notifs = await self.registration.getNotifications({ tag })
  notifs.forEach((n) => n.close())
  return  // do NOT show a new notification
}
```

### Native (Capacitor / Android)

`frontend/src/shared/earlyPush.ts` registers `pushNotificationReceived` at
app boot (must be early or the cold-start tap event gets dropped). Branches
on `notif.data.type === 'dismiss'`:

```ts
const { notifications } = await PushNotifications.getDeliveredNotifications()
const tag = notif.data.tag || `chat-${notif.data.chat_id}`
const toRemove = notifications.filter((n) => n.tag === tag)
if (toRemove.length) {
  await PushNotifications.removeDeliveredNotifications({ notifications: toRemove })
}
```

### Local dismiss on chat-enter (every device)

`shared/dismissChatNotifications.ts` is called from `ChatsPage` whenever
`activeChatId` changes, alongside the existing `mark_read` WS send. It
performs the same `getNotifications({tag}) + close` locally so the user
doesn't see their own tray entry sit there for a round-trip while waiting
for the dismiss-push to bounce back.

## Known limitations

| Case | What happens | Why |
|---|---|---|
| Android app **Force-stopped** via Settings | Tray entry persists until the user next opens the app | FCM doesn't deliver data-only messages to force-stopped processes — Android OS rule. Same trade-off Telegram / WhatsApp / Slack have. Once the user opens the app, the chat-enter hook + WS reconnect catch up. |
| Browser closed entirely (no SW process) | Dismiss delivers on next browser open / push wake | Standard web push lifecycle. |
| Partial read (1 of N visible) | Tray entry **stays** because backend still sees `unread > 0` | By design — collapsed tray is a per-chat indicator, not per-message. Open the chat (mark_read empties everything) to clear it. |
| Race: dismiss-push arrives **after** a fresh new-message push for the same chat | Tray briefly empty, then re-appears with the new message | Correct. The dismiss only closes the entries that exist at that moment; later pushes are independent. |
| Stale dismiss after long offline period (e.g. plane mode > 1 min) | Push provider drops it; tray reflects current truth, not history | Both dismiss paths set a **60s TTL** (web push HTTP header + FCM `AndroidConfig.TTL`). Defends against the scenario "dismiss issued at t=10 arrives at t=100 and closes a tray entry for a NEW message that arrived at t=50". Regular new-message pushes keep their 24h TTL — those should always deliver eventually. |
| User has multiple devices, reads on A | Devices B, C, D all clear within ~1s | Backend blasts dismiss to every subscription / FCM token of the user. The source device dismisses locally (instant) and gets the dismiss-push too (no-op since nothing to close). |
| `pushService.SendDismiss` itself errors | Logged at WARN, no retry | Best-effort. Worst case: tray entry sticks until the user manually clears or opens the chat (which would re-fire). |

## Why not just rely on tag-replacement?

Web Push spec says a new push with the same `tag` REPLACES the visible
notification. So one could imagine "push a 'silent' push with same tag to
overwrite". But:

- Replacing requires *some* visible content. You can't push a notification
  that just hides the existing one.
- We'd have to show "✓ read" or similar, which is worse UX than just
  disappearing.

`getNotifications` + `close()` is the correct primitive. It's supported on
all evergreen browsers and on Capacitor via `removeDeliveredNotifications`.

## Operational notes

- No new DB tables, no migrations.
- No new env vars (uses existing VAPID + FCM credentials).
- Dispatch is fire-and-forget background goroutines. WS / REST request
  latency unaffected.
- Per-user cost: `O(active_subscriptions + active_fcm_tokens)` HTTP calls
  per trigger event. With current per-user limits (10 web subs + 10 FCM
  tokens) that's at most 20 push API calls per `mark_read`.
- For a 50-member group chat clear: 50 × 20 = up to 1000 push calls.
  Realistic prod groups are smaller; even the worst case completes in
  a few seconds via the existing per-call goroutines.
