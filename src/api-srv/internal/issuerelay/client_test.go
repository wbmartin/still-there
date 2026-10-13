package issuerelay

import (
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

// A file part carries the file's real type: CreateFormFile labelled every
// one application/octet-stream, so the tracker saw screenshots as bytes.
func TestMultipartBody_FilePartsCarryTheirType(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	body, contentType, err := multipartBody(
		[][2]string{{"summary", "Save does nothing"}},
		[]File{
			{Filename: `shot "1".png`, Content: png},
			{Filename: "run.log", ContentType: "text/plain", Content: []byte("x")},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(strings.NewReader(body.String()), params["boundary"])
	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	fileList := form.File["files"]
	if len(fileList) != 2 {
		t.Fatalf("want 2 file parts, got %d", len(fileList))
	}
	if got := fileList[0].Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("sniffed type: got %q, want image/png", got)
	}
	if fileList[0].Filename != `shot "1".png` {
		t.Errorf("filename: got %q", fileList[0].Filename)
	}
	if got := fileList[1].Header.Get("Content-Type"); got != "text/plain" {
		t.Errorf("given type: got %q, want text/plain", got)
	}
}
