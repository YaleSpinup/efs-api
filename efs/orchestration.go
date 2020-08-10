package efs

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
)

// TODO centralize this retry logic
type stop struct {
	error
}

// retry is stolen from https://upgear.io/blog/simple-golang-retry-function/
func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			time.Sleep(sleep)
			return retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}

func (e *EFS) DeleteFileSystemO(fs string) error {
	ctx := context.Background()

	mounttargets, err := e.ListMountTargetsForFileSystem(ctx, fs)
	if err != nil {
		return err
	}

	for _, mt := range mounttargets {
		if status := aws.StringValue(mt.LifeCycleState); status != "available" {
			return fmt.Errorf("filesystem %s mount target %s has status %s, cannot delete", fs, aws.StringValue(mt.MountTargetId), status)
		}

		if err := e.DeleteMountTarget(ctx, aws.StringValue(mt.MountTargetId)); err != nil {
			return err
		}
	}

	if err = retry(10, 2*time.Second, func() error {
		out, err := e.GetFileSystem(ctx, fs)
		if err != nil {
			log.Warnf("error getting filesystem %s during delete: %s", fs, err)
			return err
		}

		if num := aws.Int64Value(out.NumberOfMountTargets); num > 0 {
			log.Warnf("number of mount targets for filesystem %s > 0 (current: %d)", fs, num)
			return fmt.Errorf("waiting for number of mount targets for filesystem %s to be 0 (current: %d)", fs, num)
		}

		return nil
	}); err != nil {
		return err
	}

	if err = retry(3, 2*time.Second, func() error {
		if err := e.DeleteFileSystem(ctx, fs); err != nil {
			log.Warnf("error deleting filesystem %s: %s", fs, err)
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
