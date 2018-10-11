package blobstore

import (
	"context"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func convertS3Error(err error) error {
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case s3.ErrCodeNoSuchKey, "NotFound":
				err = status.Errorf(codes.NotFound, awsErr.Message())
			}
		}
	}
	return err
}

type s3BlobAccess struct {
	s3         *s3.S3
	uploader   *s3manager.Uploader
	bucketName *string
	blobKeyer  util.DigestKeyer
}

// NewS3BlobAccess creates a BlobAccess that uses an S3 bucket as its backing
// store.
func NewS3BlobAccess(s3 *s3.S3, uploader *s3manager.Uploader, bucketName, keyPrefix *string, blobKeyer util.DigestKeyer) BlobAccess {
	return &s3BlobAccess{
		s3:         s3,
		uploader:   uploader,
		bucketName: bucketName,
		blobKeyer:  prefixedKeyer(*keyPrefix, blobKeyer),
	}
}

func prefixedKeyer(keyPrefix string, underlyingKeyer util.DigestKeyer) util.DigestKeyer {
	if len(keyPrefix) == 0 {
		return underlyingKeyer
	}
	if keyPrefix[len(keyPrefix)-1] != '/' {
		keyPrefix += "/"
	}
	return func(instance string, digest *remoteexecution.Digest) (string, error) {
		key, err := underlyingKeyer(instance, digest)
		if err == nil {
			return keyPrefix + key, nil
		}
		return key, err
	}
}

func (ba *s3BlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return util.NewErrorReader(err)
	}
	result, err := ba.s3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: ba.bucketName,
		Key:    &key,
	})
	if err != nil {
		return util.NewErrorReader(convertS3Error(err))
	}
	return result.Body
}

func (ba *s3BlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	defer r.Close()
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return err
	}
	_, err = ba.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: ba.bucketName,
		Key:    &key,
		Body:   r,
	})
	return convertS3Error(err)
}

func (ba *s3BlobAccess) Delete(ctx context.Context, instance string, digest *remoteexecution.Digest) error {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return err
	}
	_, err = ba.s3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: ba.bucketName,
		Key:    &key,
	})
	return convertS3Error(err)
}

func (ba *s3BlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	for _, digest := range digests {
		key, err := ba.blobKeyer(instance, digest)
		if err != nil {
			return nil, err
		}
		_, err = ba.s3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
			Bucket: ba.bucketName,
			Key:    &key,
		})
		if err != nil {
			err = convertS3Error(err)
			if status.Code(err) == codes.NotFound {
				missing = append(missing, digest)
			} else {
				return nil, err
			}
		}
	}
	return missing, nil
}
