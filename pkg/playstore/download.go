package playstore

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DownloadProgress struct {
	DownloadedSize int64
	DownloadSize   int64
	Err            error
}

func downloadFileToDisk(url string, downloadSize int64, sha1Checksum []byte, filepath string) (chan DownloadProgress, error) {
	const maxChunkSize = 1 * 1000 * 1000 // 1 MB
	buffer := make([]byte, maxChunkSize)

	downloadedSize := int64(0)

	data, err := downloadFile(url)
	if err != nil {
		return nil, err
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}

	h := sha1.New()

	ch := make(chan DownloadProgress)
	sendDownloadProgress := func(err error) {
		ch <- DownloadProgress{
			DownloadedSize: downloadedSize,
			DownloadSize:   downloadSize,
			Err:            err,
		}
	}

	// Delete the file, if the download was interrupted
	deleteDownload := func() {
		_ = os.Remove(filepath)
	}

	go func() {
		sendDownloadProgress(nil)

		for downloadedSize < downloadSize {
			chunkSize := downloadSize - downloadedSize
			if chunkSize > maxChunkSize {
				chunkSize = maxChunkSize
			}

			nRead, err := io.ReadFull(data, buffer[:chunkSize])
			if err != nil {
				deleteDownload()
				sendDownloadProgress(err)
				break
			}

			downloadedSize += int64(nRead)

			h.Write(buffer[:nRead])

			_, err = out.Write(buffer[:nRead])
			if err != nil {
				deleteDownload()
				sendDownloadProgress(err)
				break
			}

			sendDownloadProgress(nil)
		}

		if downloadedSize == downloadSize && !bytes.Equal(h.Sum(nil), sha1Checksum) {
			sendDownloadProgress(fmt.Errorf("checksum mismatch"))
		}
		close(ch)

		_ = out.Close()
		_ = data.Close()
	}()
	return ch, nil
}


// DownloadFile downloads a file and write it to disk during download
// https://golangcode.com/download-a-file-from-a-url/
func downloadFile(url string) (io.ReadCloser, error) {
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
