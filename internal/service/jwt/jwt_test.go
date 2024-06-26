package jwt

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/common"
)

func TestJWTManager(t *testing.T) {
	// Setup
	logger.SetGlobalLogger(&config.Config{
		LogLevel:    "info",
		ServiceName: "test",
	})

	fs := fstest.MapFS{
		"fixtures": {
			Mode: fs.ModeDir,
		},
		"fixtures/private.pem": {
			Data: []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAtq3O+BNR3tZKE+nlv10cZrF6bJ4ckcQf5ZGYvksyweyXXgwW
RFdxbgugOmf0FXQ0jhYzP1nzI0fB2K/u2cFJLI10tL+bTE86pnOcMDoyeCZh4lq5
MsQhbxNzo95Rg5vEDvghT631btC9sQoLi0qvqvREd2hA0DMtMlDifGjRGTCDILL9
OvfAvZBKziykvOghOn3tNWYsQgz2LPHi51BbCGB4IPJhBVd74HTKjIUF/Lxk7Pm0
labCxdAu8OKP6W+X9uSh7UskOkBKtHSHJVCZDJvNymSeGkQgIWcygBcA9nME7qXZ
y/FhcfAGvNAQFGCxhrfvfjC2H9FR3VolcdW2rQIDAQABAoIBACitQRXKL4PAEZSn
k2nuEMHpKQqAlnn6wuN6bRmKYw39YaMma9rh4bcQTahTt20DiCRPWy/zFom3k7lt
S3EfcezCvsb0l18BkVy5B4FRpCVO3qLpcq2UMKGsIibN/Tah+6EdrUUxxiHbxzFh
vDpS8hTN+WThSPVTP/AhRJ1RNaY236NdVEgdKWI+dM1t/Nty8XagJDBuT/JY8bpC
eFDWQ1qEnlDB9y7TgXlyb19CUPqMY/Ebm3dVxgzSWOP3bqE/V6tWUh7knxljKECN
BpYCnTHS1IFQ4pR3cu5r09l+G4GMnlP+fKDOe3jvdyetifSjMcAbXHyfpmiZGUQe
RJz6C30CgYEA55eNTkyrF1yl7VlMGx01wMcFwIAAKU1OMiRiiqjRs3bbOTz30pEY
eVKpK6tyfZBsxLeKlGL2lD5Skg4yo8ckcZ8scAGmkVB2LKFnk9+lX7kgWm/eAM42
5XuN81LfgDNWtb6n9AJ2agMbqphEleRot5l6HXLcCsLLDzLfC6dfgScCgYEAye6P
1xQGo+FvkM5QMQsUMjdXEyyKFxVb4GD4siCXhtZyMn+zMag2g5k3ncwCVjxlQ82E
8FM5wxZr2OA5vp3LWly8pI7Yw2+isBrIMr2oOHV2QJCn15FPQK9O74t95tTTlGHr
407xZGIPP8GXkhPPYDH0QPL/Jftxe8fxcr7RxgsCgYAODcNUchCb3VJwYc/dgVtG
tI0jzmC0IO3S2yRjt7TqCBdrlMiRLZ7nld2QOdo7xmzjTyQItyyxeEq4dEYcbDRI
9NjUfzUlclWJhc3sSlEVyv0sn8dAE0N/j4zgrDHF7NehNc2pYBDhhAjExHK9CdxU
7+paKSMzP/jklji001ZXVwKBgHrqL7wngHM4wgRO0RlJOR3n+aS+M8AhTC+kVz12
AUYeOpzqhlTvo18vYF840yNS2AERlJ4EyuApQbRdqEiTHDkAwgMYwHEV/t1bMAlS
0JatSTG7266n0Kn7C/1b12MuoSts/3z5jI4h8k5ItM5CKLTRM3BleVHRYB6Mcjf6
Vw5JAoGAQfwwRLabE42KfmE5TuWF//QqHqEcpH7aa1f4plh/BlhshRoNImGmE5q9
95AXNK7lMJ/pYGxzX3kZDIKuoROQBdeUe2spu/rP5B7DjcdBzSgQ/j76cb9fqnEn
BXRCtru+T/KYsxgu8LncAmsKDC1r7SSOx19rqoD21gzTC/1Xn2w=
-----END RSA PRIVATE KEY-----`),
		},
		"fixtures/public.pem": {
			Data: []byte(`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtq3O+BNR3tZKE+nlv10c
ZrF6bJ4ckcQf5ZGYvksyweyXXgwWRFdxbgugOmf0FXQ0jhYzP1nzI0fB2K/u2cFJ
LI10tL+bTE86pnOcMDoyeCZh4lq5MsQhbxNzo95Rg5vEDvghT631btC9sQoLi0qv
qvREd2hA0DMtMlDifGjRGTCDILL9OvfAvZBKziykvOghOn3tNWYsQgz2LPHi51Bb
CGB4IPJhBVd74HTKjIUF/Lxk7Pm0labCxdAu8OKP6W+X9uSh7UskOkBKtHSHJVCZ
DJvNymSeGkQgIWcygBcA9nME7qXZy/FhcfAGvNAQFGCxhrfvfjC2H9FR3VolcdW2
rQIDAQAB
-----END PUBLIC KEY-----`),
		},
	}
	SetupJWTManager("fixtures/public.pem", "fixtures/private.pem", fs)

	// Test case 1
	traceId, spanId := uuid.New().String(), uuid.New().String()
	ctx := context.WithValue(
		context.WithValue(
			context.Background(),
			common.ContextKeyTraceId, traceId,
		),
		common.ContextKeySpanId, spanId,
	)
	account := "john.doe@example.com"
	tokenString, err := GetJWTManager().BuildAndSignToken(ctx, account)
	assert.NoError(t, err)
	valid, err := GetJWTManager().ParseAndValidateToken(ctx, account, tokenString)
	assert.NoError(t, err)
	assert.Equal(t, true, valid)

	// Add more test cases if needed
}
