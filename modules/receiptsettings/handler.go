package receiptsettings

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo       *Repository
	uploadsDir string
	baseURL    string
}

func NewHandler(repo *Repository, uploadsDir, baseURL string) *Handler {
	return &Handler{repo: repo, uploadsDir: uploadsDir, baseURL: baseURL}
}

// GetSettings godoc
// @Summary      Get receipt settings
// @Description  Returns the current receipt print settings (store name, footer, template, etc.)
// @Tags         ReceiptSettings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=receiptsettings.ReceiptSettings}
// @Failure      401  {object}  response.APIResponse
// @Router       /api/receipt-settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	s, err := h.repo.Get(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, s)
}

// UpdateSettings godoc
// @Summary      Update receipt settings
// @Description  Updates receipt print settings; validates custom Go template syntax if provided
// @Tags         ReceiptSettings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  UpdateRequest  true  "Receipt settings to update"
// @Success      200  {object}  response.APIResponse{data=receiptsettings.ReceiptSettings}
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /api/receipt-settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	// Validate custom template syntax if provided
	if req.Template != nil && *req.Template != "" {
		if _, err := template.New("receipt").Parse(*req.Template); err != nil {
			c.Error(apperr.Validation("invalid template syntax: " + err.Error()))
			return
		}
	}

	if err := h.repo.Update(c.Request.Context(), req); err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	// Return merged settings
	s, err := h.repo.Get(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, s)
}

// UploadLogo godoc
// @Summary      Upload receipt logo
// @Description  Upload a logo image for receipts (jpg/jpeg/png/webp, max 2MB)
// @Tags         ReceiptSettings
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file  formData  file  true  "Logo image (jpg/jpeg/png/webp, max 2MB)"
// @Success      200   {object}  response.APIResponse{data=ReceiptSettings}
// @Failure      400   {object}  response.APIResponse
// @Router       /api/settings/receipt/logo [post]
func (h *Handler) UploadLogo(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.Error(apperr.Validation("file is required"))
		return
	}
	if fh.Size > 2*1024*1024 {
		c.Error(apperr.Validation("file too large (max 2MB)"))
		return
	}

	// Validate content type
	src, err := fh.Open()
	if err != nil {
		c.Error(apperr.Internal(fmt.Errorf("open upload: %w", err)))
		return
	}
	defer src.Close()

	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		c.Error(apperr.Internal(fmt.Errorf("read upload: %w", err)))
		return
	}
	ct := http.DetectContentType(buf[:n])

	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowed[ct] {
		c.Error(apperr.Validation("unsupported file type, allowed: jpg, jpeg, png, webp"))
		return
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	allowedExt := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExt[ext] {
		c.Error(apperr.Validation("unsupported file extension"))
		return
	}

	// Save file
	storedPath := "logo" + ext
	fullPath := filepath.Join(h.uploadsDir, storedPath)

	// Security check
	absUploads, _ := filepath.Abs(h.uploadsDir)
	absFile, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFile, absUploads) {
		c.Error(apperr.Validation("invalid file path"))
		return
	}

	if err := os.MkdirAll(h.uploadsDir, 0755); err != nil {
		c.Error(apperr.Internal(fmt.Errorf("create upload dir: %w", err)))
		return
	}

	// Delete old logo (best-effort)
	ctx := c.Request.Context()
	old, _ := h.repo.Get(ctx)
	if old.LogoPath != "" {
		_ = os.Remove(filepath.Join(h.uploadsDir, old.LogoPath))
	}

	// Seek back
	if seeker, ok := src.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	} else {
		src.Close()
		src, err = fh.Open()
		if err != nil {
			c.Error(apperr.Internal(fmt.Errorf("reopen upload: %w", err)))
			return
		}
		defer src.Close()
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		c.Error(apperr.Internal(fmt.Errorf("create file: %w", err)))
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		c.Error(apperr.Internal(fmt.Errorf("write file: %w", err)))
		return
	}

	// Update DB
	if err := h.repo.Update(ctx, UpdateRequest{LogoPath: &storedPath}); err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	s, err := h.repo.Get(ctx)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	var logoURL string
	if s.LogoPath != "" && h.baseURL != "" {
		logoURL = h.baseURL + "/uploads/" + s.LogoPath
	}

	response.OK(c, gin.H{
		"settings": s,
		"logo_url": logoURL,
	})
}
