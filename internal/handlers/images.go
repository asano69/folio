package handlers

import (
	"net/http"
	"folio/internal/library"
	"path/filepath"
	"strings"
)

type ImageHandler struct {
	Library *library.Library
}

func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// URLパスから /images/ プレフィックスを除去
	imagePath := strings.TrimPrefix(r.URL.Path, "/images/")

	// パストラバーサル攻撃を防ぐ
	if strings.Contains(imagePath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// 拡張子チェック
	ext := strings.ToLower(filepath.Ext(imagePath))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	// 実際のファイルパスを構築
	fullPath := filepath.Join(h.Library.Path, imagePath)

	// ファイルを配信
	http.ServeFile(w, r, fullPath)
}
