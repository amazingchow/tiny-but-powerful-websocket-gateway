package utils

import (
	"time"

	"github.com/juju/ratelimit"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
)

type TPSController struct {
	quota  int64
	bucket *ratelimit.Bucket
}

func NewTPSController(quota int64) *TPSController {
	ctrl := &TPSController{quota: quota}
	ctrl.bucket = ratelimit.NewBucket(time.Duration(1000.0/float64(quota))*(time.Millisecond), quota)
	return ctrl
}

// TakeToken takes a token from the bucket, if the bucket is full, it will wait until the resource turns to be available.
func (ctrl *TPSController) TakeToken() {
	waitUntilAvailable := ctrl.bucket.Take(1)
	if waitUntilAvailable != 0 {
		logger.GetGlobalLogger().Warningf("TPS-Quota-Limit(%d) exceeds, wait %s secs until the resource turns to be available.",
			ctrl.quota, waitUntilAvailable.String())
		time.Sleep(waitUntilAvailable)
	}
}
