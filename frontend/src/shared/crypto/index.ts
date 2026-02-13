// E2E Encryption public API
// Re-export everything consumers need

export { encryptText, decryptText, encryptFile, decryptFile } from './encryption'
export {
  getOrDeriveChatKey,
  initializeKeys,
  backupPrivateKey,
  restorePrivateKey,
} from './keyExchange'
export { hasIdentityKeys, clearAllCryptoData } from './keyStore'
