package service

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

type UtilService struct{}

func NewUtilService() *UtilService {
	return &UtilService{}
}

func (s *UtilService) Now() (sec, ms int64, rfc string) {
	now := time.Now()
	return now.Unix(), now.UnixMilli(), now.Format(time.RFC3339)
}

func (s *UtilService) ParseTS(ts string) (sec, ms int64, rfc string, err error) {
	v, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return 0, 0, "", err
	}
	var t time.Time
	if v > 1e11 {
		t = time.UnixMilli(v)
	} else {
		t = time.Unix(v, 0)
	}
	return t.Unix(), t.UnixMilli(), t.Format(time.RFC3339), nil
}

// GenUUID 生成 n 个符合 RFC 4122 的 UUID v4
func (s *UtilService) GenUUID(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = uuid.New().String()
	}
	return out
}
