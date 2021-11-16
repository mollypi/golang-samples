package storagetransfer

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	storagetransfer "cloud.google.com/go/storagetransfer/apiv1"
	storagetransferpb "google.golang.org/genproto/googleapis/storagetransfer/v1"

	"github.com/GoogleCloudPlatform/golang-samples/internal/testutil"
)

func TestQuickstart(t *testing.T) {
	tc := testutil.SystemTest(t)
	ctx := context.Background()

	str, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}
	defer str.Close()

	sts, err := storagetransfer.NewClient(ctx)
	if err != nil {
		log.Fatalf("storagetransfer.NewClient: %v", err)
	}
	defer sts.Close()

	sinkBucketName, err := testutil.CreateTestBucket(ctx, t, str, tc.ProjectID, "sts-go-sink")
	if err != nil {
		t.Fatalf("Bucket creation failed: %v", err)
	}
	defer testutil.DeleteBucketIfExists(ctx, str, sinkBucketName)

	sourceBucketName, err := testutil.CreateTestBucket(ctx, t, str, tc.ProjectID, "sts-go-source")
	if err != nil {
		t.Fatalf("Bucket creation failed: %v", err)
	}
	defer testutil.DeleteBucketIfExists(ctx, str, sourceBucketName)

	grantSTSPermissions(sinkBucketName, tc.ProjectID, sts, str)
	grantSTSPermissions(sourceBucketName, tc.ProjectID, sts, str)

	buf := new(bytes.Buffer)
	resp, err := quickstart(buf, tc.ProjectID, sourceBucketName, sinkBucketName)

	if err != nil {
		t.Errorf("quickstart: %#v", err)
	}

	got := buf.String()
	if want := "transferJobs/"; !strings.Contains(got, want) {
		t.Errorf("quickstart: got %q, want %q", got, want)
	}

	tj := &storagetransferpb.TransferJob{
		Name: resp.Name,
		Status: storagetransferpb.TransferJob_DELETED,
	}
	sts.UpdateTransferJob(ctx, &storagetransferpb.UpdateTransferJobRequest{
		JobName: resp.Name,
		ProjectId: tc.ProjectID,
		TransferJob: tj,
	})
}

func grantSTSPermissions(bucketName string, projectID string, sts *storagetransfer.Client, str *storage.Client) {
	ctx := context.Background()

	req := &storagetransferpb.GetGoogleServiceAccountRequest{
		ProjectId: projectID,
	}

	resp, err := sts.GetGoogleServiceAccount(ctx, req)
	if err != nil {
		fmt.Print("Error getting service account")
	}
	email := resp.AccountEmail

	identity := "serviceAccount:" + email

	bucket := str.Bucket(bucketName)
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		log.Fatalf("Bucket(%q).IAM().Policy: %v", bucketName, err)
	}

	var objectViewer iam.RoleName = "roles/storage.objectViewer"
	var bucketReader iam.RoleName = "roles/storage.legacyBucketReader"
	var bucketWriter iam.RoleName = "roles/storage.legacyBucketWriter"

	policy.Add(identity, objectViewer)
	policy.Add(identity, bucketReader)
	policy.Add(identity, bucketWriter)

	if err := bucket.IAM().SetPolicy(ctx, policy); err != nil {
		log.Fatalf("Bucket(%q).IAM().SetPolicy: %v", bucketName, err)
	}
}
