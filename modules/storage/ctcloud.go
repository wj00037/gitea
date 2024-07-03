package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/minio/minio-go/v7"
)

var ct_default_expire = 15 * time.Minute

// NewCTCloudStorage returns a ctcloud storage
func NewCTCloudStorage(ctx context.Context, cfg *setting.Storage) (ObjectStorage, error) {
	obsCfg := &cfg.MinioConfig

	conf := &aws.Config{
		Endpoint:         aws.String(obsCfg.Endpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(obsCfg.AccessKeyID, obsCfg.SecretAccessKey, ""),
		Region:           aws.String(obsCfg.Location),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{Config: *conf}))
	cli := s3.New(sess)

	m, err := NewMinioStorage(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &CTCloudStorage{
		ctclient:     cli,
		bucketDomain: cfg.MinioConfig.BucketDomain,
		MinioStorage: m.(*MinioStorage),
	}, nil
}

type CTCloudStorage struct {
	ctclient     *s3.S3
	bucketDomain string

	*MinioStorage
}

func (ctc *CTCloudStorage) GenerateMultipartParts(path string, size int64) (parts []*structs.MultipartObjectPart, abort *structs.MultipartEndpoint, verify *structs.MultipartEndpoint, err error) {
	objectKey := ctc.buildMinioPath(path)
	taskParts := map[int64]s3.Part{}
	uploadID := ""
	//1. list all the multipart tasks
	listMultipart := &s3.ListMultipartUploadsInput{}
	listMultipart.Prefix = &objectKey
	listMultipart.Bucket = &ctc.bucket
	listResult, err := ctc.ctclient.ListMultipartUploads(listMultipart)
	if err != nil {
		log.Error("lfs[multipart] Failed to list existing multipart task %s and %s", ctc.bucket, objectKey)
		return nil, nil, nil, err
	}
	if len(listResult.Uploads) != 0 {
		//remove all unfinished tasks if multiple tasks are found
		if len(listResult.Uploads) > 1 {
			for _, task := range listResult.Uploads {
				abortRequest := &s3.AbortMultipartUploadInput{}
				abortRequest.Key = &objectKey
				abortRequest.Bucket = &ctc.bucket
				abortRequest.UploadId = task.UploadId
				_, err = ctc.ctclient.AbortMultipartUpload(abortRequest)
				if err != nil {
					log.Error("lfs[multipart] Failed to abort existing multipart task %s and %s %s", ctc.bucket, objectKey, task.UploadId)
					return nil, nil, nil, err
				}
			}
		} else {
			//find out all finished tasks
			partRequest := &s3.ListPartsInput{}
			partRequest.Key = &objectKey
			partRequest.Bucket = &ctc.bucket
			partRequest.UploadId = listResult.Uploads[0].UploadId
			uploadID = *listResult.Uploads[0].UploadId
			parts, err := ctc.ctclient.ListParts(partRequest)
			if err != nil {
				log.Error("lfs[multipart] Failed to get existing multipart task part %s and %s %s", ctc.bucket, objectKey)
			}
			for _, content := range parts.Parts {
				taskParts[int64(*content.PartNumber)] = *content
			}
		}
	}
	//2. get and return all unfinished tasks, clean up the task if needed
	//TODO
	//3. Initialize multipart task
	if uploadID == "" {
		log.Trace("lfs[multipart] Starting to create multipart task %s and %s", ctc.bucket, objectKey)
		upload := s3.CreateMultipartUploadInput{}
		minioPath := ctc.buildMinioPath(path)
		upload.Key = &minioPath
		upload.Bucket = &ctc.bucket
		multipart, err := ctc.ctclient.CreateMultipartUpload(&upload)
		uploadID = *multipart.UploadId
		if err != nil {
			return nil, nil, nil, err
		}
	}
	//generate part
	currentPart := int64(0)
	for {
		if currentPart*multipart_chunk_size >= size {
			break
		}
		partSize := size - currentPart*multipart_chunk_size
		if partSize > multipart_chunk_size {
			partSize = multipart_chunk_size
		}
		//check part exists and length matches
		if value, existed := taskParts[currentPart+1]; existed {
			if value.Size == &partSize {
				log.Trace("lfs[multipart] Found existing part %d for multipart task %s and %s, will add etag information", currentPart+1, ctc.bucket, objectKey)
				var part = &structs.MultipartObjectPart{
					Index:             int(currentPart) + 1,
					Pos:               currentPart * multipart_chunk_size,
					Size:              partSize,
					Etag:              strings.Trim(*value.ETag, "\""),
					MultipartEndpoint: nil,
				}
				parts = append(parts, part)
				currentPart += 1
				continue
			} else {
				log.Trace("lfs[multipart] Found existing part %d while size not matched for multipart task %s and %s", currentPart+1, ctc.bucket, objectKey)
			}
		}
		input := &s3.PutObjectInput{
			Bucket: &ctc.bucket,
			Key:    &objectKey,
		}
		request, _ := ctc.ctclient.PutObjectRequest(input)
		output, err := request.Presign(ct_default_expire)
		if err != nil {
			return nil, nil, nil, err
		}
		var part = &structs.MultipartObjectPart{
			Index: int(currentPart) + 1,
			Pos:   currentPart * multipart_chunk_size,
			Size:  partSize,
			MultipartEndpoint: &structs.MultipartEndpoint{
				ExpiresIn:         default_expire,
				Href:              output,
				Method:            http.MethodPut,
				Headers:           nil,
				Params:            nil,
				AggregationParams: nil,
			},
		}
		parts = append(parts, part)
		currentPart += 1
	}
	//generate abort
	//TODO
	//generate verify
	verify = &structs.MultipartEndpoint{
		Params: &map[string]string{
			"upload_id": uploadID,
		},
		AggregationParams: &map[string]string{
			"key":  "part_ids",
			"type": "array",
			"item": "index,etag",
		},
	}
	return parts, nil, verify, nil
}

func (ctc *CTCloudStorage) CommitUpload(path, additionalParameter string) error {
	var param MultiPartCommitUpload
	err := json.Unmarshal([]byte(additionalParameter), &param)
	if err != nil {
		log.Error("lfs[multipart] unable to decode additional parameter", additionalParameter)
		return err
	}
	if len(param.UploadID) == 0 || len(param.PartIDs) == 0 {
		log.Error("lfs[multipart] failed to commit objects, parameter is empty %v", param)
		return errors.New("parameter is empty")
	}
	log.Trace("lfs[multipart] start to commit upload object %v", param)
	//merge multipart
	parts := make([]*s3.CompletedPart, 0, len(param.PartIDs))
	for _, p := range param.PartIDs {
		index := int64(p.Index)
		part := s3.CompletedPart{ETag: &p.Etag, PartNumber: &index}
		parts = append(parts, &part)
	}
	complete := &s3.CompleteMultipartUploadInput{}
	complete.Bucket = &ctc.bucket
	key := ctc.buildMinioPath(path)
	complete.Key = &key
	complete.UploadId = &param.UploadID
	complete.MultipartUpload = &s3.CompletedMultipartUpload{Parts: parts}
	log.Trace("lfs[multipart] Start to merge multipart task %s and %s", ctc.bucket, ctc.buildMinioPath(path))
	_, err = ctc.ctclient.CompleteMultipartUpload(complete)
	if err != nil {
		// handle the case if task with identical object has been committed before, return nil and let obs storage check the existence of object.
		// 如果请求的多段上传任务不存在，AWS返回404 Not Found，包含错误信息NoSuchUpload
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NoSuchUpload" {
			log.Trace("lfs[multipart] unable to complete multipart task %s and %s, unable to find upload task, maybe "+
				"it completed just now.", ctc.bucket, ctc.buildMinioPath(path))
			return nil
		}
		return err
	}
	//TODO notify CDN to fetch new object
	return nil

}

// URL gets the redirect URL to download a file. The presigned link is valid for 5 minutes.
func (ctc *CTCloudStorage) URL(path, name string) (*url.URL, error) {
	//NOTE: we url.PathEscape instead of url.QueryEscape is used here due to we need to convert space to %20 rather than +
	// queryParameter := map[string]string{"response-content-disposition": "attachment; filename=\"" + url.PathEscape(name) + "\""}
	input := &s3.GetObjectInput{
		Bucket: aws.String(ctc.bucket),
		Key:    aws.String(ctc.buildMinioPath(path)),
	}

	request, _ := ctc.ctclient.GetObjectRequest(input)
	output, err := request.Presign(ct_default_expire)
	if err != nil {
		return nil, err
	}

	//NOTE: it will work since CDN will replace hostname back to obs domain and that will make signed url work.
	v, err := url.Parse(output)
	if err == nil {
		v.Host = ctc.bucketDomain
		v.Scheme = "https"
	}

	return v, err
}

// IterateObjectsKeyOnly iterates across the objects' name only in the miniostorage
func (ctc *CTCloudStorage) IterateObjectsKeyOnly(path string, fn func(path string) error) error {
	for mObjInfo := range ctc.client.ListObjects(ctc.ctx, ctc.bucket, minio.ListObjectsOptions{
		Prefix:    "",
		Recursive: true,
		MaxKeys:   500,
	}) {
		return fn(mObjInfo.Key)
	}
	return nil
}

func init() {
	RegisterStorageType(setting.CTCloudStorageType, NewMinioStorage)
}
