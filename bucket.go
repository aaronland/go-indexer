package indexer

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/aaronland/gocloud-blob/bucket"
	"gocloud.dev/blob"
)

// START OF put me in aaronland/gocloud-blob

func deriveBucketURIAndKey(ctx context.Context, uri string) (string, string, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return "", "", fmt.Errorf("Failed to parse archive URI, %w", err)
	}

	key := filepath.Base(u.Path)
	u.Path = filepath.Dir(u.Path)

	bucket_uri := u.String()
	return bucket_uri, key, nil
}

func deriveBucketAndKey(ctx context.Context, uri string) (*blob.Bucket, string, error) {

	bucket_uri, key, err := deriveBucketURIAndKey(ctx, uri)

	if err != nil {
		return nil, "", fmt.Errorf("Failed to parse archive URI, %w", err)
	}

	b, err := bucket.OpenBucket(ctx, bucket_uri)

	if err != nil {
		return nil, "", fmt.Errorf("Failed to open bucket (%s) derived from archive URI, %w", uri, err)
	}

	return b, key, nil
}

// END OF put me in aaronland/gocloud-blob
