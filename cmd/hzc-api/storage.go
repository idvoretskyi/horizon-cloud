package main

import (
	"bytes"
	"encoding/base64"
	"sort"
	"strings"
	"time"

	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/types"

	"google.golang.org/cloud/storage"
)

func requestsForFilelist(
	ctx *hzhttp.Context,
	client *storage.Client,
	bucket, prefix string,
	files []types.FileDescription) ([]types.FileUploadRequest, error) {

	conf := ctx.ServiceAccount()
	bucketH := client.Bucket(bucket)

	requests := make([]types.FileUploadRequest, 0, 8)

	// Look for files that are missing from the bucket or changed (should be uploaded)
	for _, file := range files {
		obj := bucketH.Object(prefix + file.Path)
		attrs, err := obj.Attrs(nil)
		if err == storage.ErrObjectNotExist {
			attrs, err = nil, nil
		}
		if err != nil {
			return nil, err
		}

		if attrs != nil && bytes.Equal(attrs.MD5, file.MD5) && attrs.ContentType == file.ContentType {
			continue
		}

		md5base64 := base64.StdEncoding.EncodeToString(file.MD5)

		signedURL, err := storage.SignedURL(bucket, prefix+file.Path, &storage.SignedURLOptions{
			GoogleAccessID: conf.Email,
			PrivateKey:     conf.PrivateKey,
			ContentType:    file.ContentType,
			Method:         "PUT",
			Expires:        time.Now().Add(15 * time.Minute),
			MD5:            []byte(md5base64),
			Headers: []string{
				"x-goog-acl:public-read\n",
			},
		})
		if err != nil {
			return nil, err
		}

		requests = append(requests, types.FileUploadRequest{
			SourcePath: file.Path,
			Method:     "PUT",
			URL:        signedURL,
			Headers: map[string]string{
				"Content-Type":  file.ContentType,
				"Cache-Control": "private,no-cache",
				"Content-MD5":   md5base64,
				"x-goog-acl":    "public-read",
			},
		})
	}

	// Look for files that exist but are not in the file list (should be deleted)
	filesInManifest := make(map[string]struct{}, len(files))
	for _, file := range files {
		filesInManifest[file.Path] = struct{}{}
	}

	listQ := &storage.Query{Prefix: prefix}
	for listQ != nil {
		list, err := bucketH.List(nil, listQ)
		if err != nil {
			return nil, err
		}

		for _, item := range list.Results {
			innerName := strings.TrimPrefix(item.Name, prefix)
			if innerName == "" {
				continue
			}
			if strings.HasPrefix(innerName, "horizon/") {
				continue
			}
			if _, ok := filesInManifest[innerName]; !ok {
				err := bucketH.Object(item.Name).Delete(nil)
				if err != nil {
					return nil, err
				}
			}
		}

		listQ = list.Next
	}

	return requests, nil
}

type objectAttrsByName []*storage.ObjectAttrs

func (o objectAttrsByName) Len() int           { return len(o) }
func (o objectAttrsByName) Less(i, j int) bool { return o[i].Name < o[j].Name }
func (o objectAttrsByName) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }

func copyAllObjects(
	ctx *hzhttp.Context,
	client *storage.Client,
	srcBucket, srcPrefix string,
	dstBucket, dstPrefix string) error {

	ctx.Info("copying all objects in bucket %#v prefix %#v to bucket %#v prefix %#v",
		srcBucket, srcPrefix, dstBucket, dstPrefix)

	srcBucketH := client.Bucket(srcBucket)
	dstBucketH := client.Bucket(dstBucket)

	srcContents := make([]*storage.ObjectAttrs, 0, 32)
	listQ := &storage.Query{Prefix: srcPrefix}
	for listQ != nil {
		list, err := srcBucketH.List(nil, listQ)
		if err != nil {
			return err
		}

		for _, item := range list.Results {
			if item.Name == srcPrefix {
				continue
			}
			srcContents = append(srcContents, item)
		}

		listQ = list.Next
	}

	dstContents := make([]*storage.ObjectAttrs, 0, 32)
	listQ = &storage.Query{Prefix: dstPrefix}
	for listQ != nil {
		list, err := dstBucketH.List(nil, listQ)
		if err != nil {
			return err
		}

		for _, item := range list.Results {
			if item.Name == dstPrefix {
				continue
			}
			dstContents = append(dstContents, item)
		}

		listQ = list.Next
	}

	sort.Sort(objectAttrsByName(srcContents))
	sort.Sort(objectAttrsByName(dstContents))

	for len(srcContents) > 0 && len(dstContents) > 0 {
		srcF := srcContents[0]
		dstF := dstContents[0]
		srcN := strings.TrimPrefix(srcF.Name, srcPrefix)
		dstN := strings.TrimPrefix(dstF.Name, dstPrefix)

		if srcN == dstN {
			// same file

			if !bytes.Equal(srcF.MD5, dstF.MD5) {
				_, err := srcBucketH.Object(srcF.Name).CopyTo(nil, dstBucketH.Object(dstF.Name), srcF)
				if err != nil {
					return err
				}
			}

			srcContents = srcContents[1:]
			dstContents = dstContents[1:]
		} else if srcN < dstN {
			// new file

			newDestName := dstPrefix + strings.TrimPrefix(srcF.Name, srcPrefix)

			_, err := srcBucketH.Object(srcF.Name).CopyTo(nil, dstBucketH.Object(newDestName), srcF)
			if err != nil {
				return err
			}

			srcContents = srcContents[1:]
		} else if srcN > dstN {
			// destination file now nonexistent

			err := dstBucketH.Object(dstF.Name).Delete(nil)
			if err != nil {
				return err
			}

			dstContents = dstContents[1:]
		} else {
			panic("not reached")
		}
	}

	if len(srcContents) > 0 {
		// all remaining source files are new

		for _, srcF := range srcContents {
			newDestName := dstPrefix + strings.TrimPrefix(srcF.Name, srcPrefix)

			_, err := srcBucketH.Object(srcF.Name).CopyTo(nil, dstBucketH.Object(newDestName), srcF)
			if err != nil {
				return err
			}
		}
	}

	if len(dstContents) > 0 {
		// all remaining destination files are now nonexistent

		for _, dstF := range dstContents {
			err := dstBucketH.Object(dstF.Name).Delete(nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
