package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"feed/config"
)

// nowUnix 当前 Unix 秒（抽成函数便于测试注入）
func nowUnix() int64 { return time.Now().Unix() }

// 点播模块：加密基元
// 文档：docs/点播模块.md 6.1 签名 URL / 6.3 内容密钥信封加密 / 6.5 防伪造
// 本文件只放「无状态、可单测」的纯函数；Redis / DB 交互在各业务 service 里

// ErrCipherInvalid 密文解密失败（密钥不符 / 数据被篡改）
var ErrCipherInvalid = errors.New("vod cipher invalid")

// kek 主密钥：SHA-256(VOD_KEK) → 32 字节 AES-256 密钥
// 文档 6.3 的诚实边界：这防的是「数据库被 dump」，防不住「应用被攻破」（主密钥在内存里）
func kek() []byte {
	sum := sha256.Sum256([]byte(config.VodKEK))
	return sum[:]
}

// SealContentKey 信封加密：用主密钥 AES-GCM 加密内容密钥，返回 (密文, nonce)
func SealContentKey(plain []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(kek())
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, plain, nil), nonce, nil
}

// OpenContentKey 解信封：主密钥 + nonce 解出内容密钥明文
func OpenContentKey(sealed, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(kek())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, ErrCipherInvalid
	}
	return plain, nil
}

// GenContentKey 生成一把内容密钥：key(16字节) + kid(16字节)，均来自 crypto/rand
func GenContentKey() (key, kid []byte, err error) {
	key = make([]byte, 16)
	kid = make([]byte, 16)
	if _, err = rand.Read(key); err != nil {
		return nil, nil, err
	}
	if _, err = rand.Read(kid); err != nil {
		return nil, nil, err
	}
	return key, kid, nil
}

// KidHex KID 的 hex 字符串（存 content_key.key_id，manifest 中引用）
func KidHex(kid []byte) string { return hex.EncodeToString(kid) }

// HexNoPrefix 去掉 0x 前缀（MP4Box 的 XML 用 0x 前缀，这里统一转换）
func HexNoPrefix(b []byte) string { return hex.EncodeToString(b) }

// ── 签名 URL（分片级 + 覆盖变体位）──
// 文档 6.1：签名 = HMAC(secret, path + exp)；本模块按 6.5 要求把「变体位」纳入签名串，
// 使「改体位 → 签名失效」，堵住「同段换变体」的伪造路径

// segSigMsg 拼签名原文：movieId|renditionId|sessionId|segNo|variant|exp
func segSigMsg(movieID, renditionID, sessionID int64, segNo int, variant byte, exp int64) string {
	return fmt.Sprintf("%d|%d|%d|%d|%d|%d", movieID, renditionID, sessionID, segNo, variant, exp)
}

// SignSegment 生成分片签名（hex）
func SignSegment(movieID, renditionID, sessionID int64, segNo int, variant byte, exp int64) string {
	mac := hmac.New(sha256.New, []byte(config.VodURLSecret))
	mac.Write([]byte(segSigMsg(movieID, renditionID, sessionID, segNo, variant, exp)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySegmentSig 校验分片签名 + 时效；全部用常量时间比较，防时序侧信道
func VerifySegmentSig(movieID, renditionID, sessionID int64, segNo int, variant byte, exp int64, sig string) bool {
	if exp < nowUnix() { // 过期
		return false
	}
	want := SignSegment(movieID, renditionID, sessionID, segNo, variant, exp)
	return subtle.ConstantTimeCompare([]byte(want), []byte(sig)) == 1
}

// SignManifest 生成 manifest 自身签名（ticket 之外的二次校验，防伪造 ticket）
func SignManifest(sessionID int64, exp int64) string {
	mac := hmac.New(sha256.New, []byte(config.VodURLSecret))
	mac.Write([]byte("manifest|" + strconv.FormatInt(sessionID, 10) + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// ── HLS 分片按会话现加密（文档 4.1：HLS 线 per-session key，分发时按需加密）──

// EncryptHLSSegment 用会话密钥 AES-128-CBC 加密一个明文分片
// HLS AES-128 标准做法：IV 由段序号决定（不传输 IV），解密端用同样的规则
func EncryptHLSSegment(plain, key []byte, segNo int) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := hlsIV(segNo)
	// PKCS#7 填充到 16 字节块整数倍
	padded := pkcs7Pad(plain, aes.BlockSize)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
	return out, nil
}

// hlsIV HLS AES-128 的 IV：把段序号写进 16 字节大端
func hlsIV(segNo int) []byte {
	iv := make([]byte, 16)
	for i := 0; i < 8; i++ {
		iv[15-i] = byte(segNo >> (8 * i))
	}
	return iv
}

// pkcs7Pad PKCS#7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	n := blockSize - len(data)%blockSize
	pad := make([]byte, n)
	for i := range pad {
		pad[i] = byte(n)
	}
	return append(data, pad...)
}

// GenSessionKey 生成 HLS 会话密钥（16 字节），存 Redis，随会话失效
func GenSessionKey() ([]byte, error) {
	k := make([]byte, 16)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}
