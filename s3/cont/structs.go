package cont

import "github.com/openimsdk/tools/s3"

type InitiateUploadResult struct {
	// UploadID uniquely identifies the upload session for tracking and management purposes.
	UploadID string `json:"uploadID"`

	// PartSize specifies the size of each part in a multipart upload. This is relevant for breaking down large uploads into manageable pieces.
	PartSize int64 `json:"partSize"`

	// Sign contains the authentication and signature information necessary for securely uploading each part. This could include signed URLs or tokens.
	Sign *s3.AuthSignResult `json:"sign"`
}

type UploadResult struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	Key  string `json:"key"`
}
