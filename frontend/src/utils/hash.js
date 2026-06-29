import CryptoJS from 'crypto-js';
export async function sha256Hex(text) {
  return CryptoJS.SHA256(text).toString(CryptoJS.enc.Hex);
}
