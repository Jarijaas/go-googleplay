package playstore

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
)

// DownloadFile downloads a file and write it to disk during download
// https://golangcode.com/download-a-file-from-a-url/
func createDownloadReader(url string) (io.ReadCloser, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("incorrect status code: %d", resp.StatusCode)
	}
	return resp.Body, err
}

func DownloadVerifySha1(url string, downloadSize int64, checksum []byte) (io.ReadCloser, error) {
	return DownloadVerify(url, downloadSize, sha1.New(), checksum)
}

func DownloadVerifySha256(url string, downloadSize int64, checksum []byte) (io.ReadCloser, error) {
	return DownloadVerify(url, downloadSize, sha256.New(), checksum)
}

func DownloadVerify(url string, downloadSize int64, hashFunc hash.Hash, checksum []byte) (io.ReadCloser, error) {
	downloadReader, err := createDownloadReader(url)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	const maxChunkSize = 32 * 1024 // 32 KiB
	buffer := make([]byte, maxChunkSize)

	downloadedSize := int64(0)

	go func() {
		for downloadedSize < downloadSize {
			chunkSize := downloadSize - downloadedSize
			if chunkSize > maxChunkSize {
				chunkSize = maxChunkSize
			}

			nRead, err := io.ReadFull(downloadReader, buffer[:chunkSize])
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}

			hashFunc.Write(buffer[:nRead])

			_, err = pw.Write(buffer[:nRead])
			if err != nil {
				_ = pw.Close()
				return
			}
			downloadedSize += int64(nRead)
		}

		if !bytes.Equal(hashFunc.Sum(nil), checksum) {
			_ = pw.CloseWithError(fmt.Errorf("checksum mismatch"))
			return
		}
		_ = pw.Close()
	}()
	return pr, nil
}
