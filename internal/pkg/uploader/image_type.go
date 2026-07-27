package uploader

import (
	"errors"
	"net/http"

	"github.com/mlogclub/simple/common/digests"
	"github.com/mlogclub/simple/common/strs"
)

// SniffLen 嗅探所需的字节数，与 http.DetectContentType 的窗口一致。
const SniffLen = 512

// ErrImageTypeNotAllowed 图片类型不在白名单内。
var ErrImageTypeNotAllowed = errors.New("image type not allowed")

// allowedImageTypes 图片上传白名单：媒体类型 → 落盘扩展名。
//
// 这里既是「允许存什么」也是「存成什么扩展名」的唯一来源。此前扩展名由
// mime.ExtensionsByType(客户端声明的 Content-Type) 推导，等价于让上传者
// 自己决定落盘扩展名与回吐时的 Content-Type —— 声明 text/html 就能拿到
// 一个在站点同源下执行的脚本文件（存储型 XSS）。
//
// 故意不含 image/svg+xml：SVG 是可以内嵌 <script> 与事件属性的 XML 文档，
// 由本站源直接提供就等于任意脚本执行，没有安全的「只存不执行」方案。
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
	"image/bmp":  ".bmp",
}

// ResolveImageType 依据文件内容判定图片类型，返回规范化的媒体类型与落盘扩展名。
//
// 只信任嗅探结果，完全忽略客户端声明的 Content-Type。head 为文件起始字节
// （至少 SniffLen 个，不足则给出全部）。类型不在白名单内时返回
// ErrImageTypeNotAllowed。
func ResolveImageType(head []byte) (contentType, ext string, err error) {
	sniffed := http.DetectContentType(head)
	// DetectContentType 可能带参数（如 text/plain; charset=utf-8），
	// 但白名单内的图片类型都不带参数，直接查表即可。
	ext, ok := allowedImageTypes[sniffed]
	if !ok {
		return "", "", ErrImageTypeNotAllowed
	}
	return sniffed, ext, nil
}

// GenerateImageKeyWithExt 用已校验过的扩展名生成图片 key（UUID）。
func GenerateImageKeyWithExt(ext string) string {
	return generateKeyWithPrefix(imagePrefix, strs.UUID(), ext)
}

// prepareCopyImage 供各 Uploader 的 CopyImage 复用：拉取远端图片、按内容
// 判定类型、生成 key。返回的 contentType 来自嗅探而非远端响应头，避免
// 远端声明 text/html 时在本站落一个可执行的 .html。
func prepareCopyImage(originUrl string) (data []byte, contentType, key string, err error) {
	data, _, err = download(originUrl)
	if err != nil {
		return nil, "", "", err
	}
	head := data
	if len(head) > SniffLen {
		head = head[:SniffLen]
	}
	contentType, ext, err := ResolveImageType(head)
	if err != nil {
		return nil, "", "", err
	}
	return data, contentType, generateKeyWithPrefix(imagePrefix, digests.MD5Bytes(data), ext), nil
}
