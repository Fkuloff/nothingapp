import { endpoints } from './endpoints'
import { httpDelete, httpGet, httpPost, httpPut } from './httpClient'
import type { AvatarUploadResponse, GroupCreateResponse, GroupInfoResponse } from './types'

export async function createGroup(
  name: string,
  memberIds: number[],
): Promise<GroupCreateResponse> {
  return httpPost<GroupCreateResponse>(endpoints.groups.create, {
    name,
    member_ids: memberIds,
  })
}

export async function getGroupInfo(groupId: number): Promise<GroupInfoResponse> {
  return httpGet<GroupInfoResponse>(endpoints.groups.info(groupId))
}

export async function updateGroupInfo(groupId: number, name: string): Promise<void> {
  await httpPut(endpoints.groups.update(groupId), { name })
}

export async function addGroupMembers(groupId: number, userIds: number[]): Promise<void> {
  await httpPost(endpoints.groups.addMembers(groupId), { user_ids: userIds })
}

export async function removeGroupMember(groupId: number, userId: number): Promise<void> {
  await httpDelete(endpoints.groups.removeMember(groupId, userId))
}

export async function leaveGroup(groupId: number): Promise<void> {
  await httpPost(endpoints.groups.leave(groupId), {})
}

export async function changeGroupRole(
  groupId: number,
  userId: number,
  role: string,
): Promise<void> {
  await httpPut(endpoints.groups.changeRole(groupId, userId), { role })
}

export async function deleteGroup(groupId: number): Promise<void> {
  await httpDelete(endpoints.groups.remove(groupId))
}

export async function uploadGroupAvatar(
  groupId: number,
  file: File,
): Promise<AvatarUploadResponse> {
  const formData = new FormData()
  formData.append('avatar', file)
  return httpPost<AvatarUploadResponse>(endpoints.groups.uploadAvatar(groupId), formData)
}

export async function deleteGroupAvatar(groupId: number): Promise<void> {
  await httpDelete(endpoints.groups.removeAvatar(groupId))
}
