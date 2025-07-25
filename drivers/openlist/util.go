package openlist

import (
	"fmt"
	"net/http"

	"github.com/OpenListTeam/OpenList/drivers/base"
	"github.com/OpenListTeam/OpenList/internal/op"
	"github.com/OpenListTeam/OpenList/pkg/utils"
	"github.com/OpenListTeam/OpenList/server/common"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

func (d *OpenList) login() error {
	if d.Username == "" {
		return nil
	}
	var resp common.Resp[LoginResp]
	_, _, err := d.request("/auth/login", http.MethodPost, func(req *resty.Request) {
		req.SetResult(&resp).SetBody(base.Json{
			"username": d.Username,
			"password": d.Password,
		})
	})
	if err != nil {
		return err
	}
	d.Token = resp.Data.Token
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *OpenList) request(api, method string, callback base.ReqCallback, retry ...bool) ([]byte, int, error) {
	url := d.Address + "/api" + api
	req := base.RestyClient.R()
	req.SetHeader("Authorization", d.Token)
	if callback != nil {
		callback(req)
	}
	res, err := req.Execute(method, url)
	if err != nil {
		code := 0
		if res != nil {
			code = res.StatusCode()
		}
		return nil, code, err
	}
	log.Debugf("[openlist] response body: %s", res.String())
	if res.StatusCode() >= 400 {
		return nil, res.StatusCode(), fmt.Errorf("request failed, status: %s", res.Status())
	}
	code := utils.Json.Get(res.Body(), "code").ToInt()
	if code != 200 {
		if (code == 401 || code == 403) && !utils.IsBool(retry...) {
			err = d.login()
			if err != nil {
				return nil, code, err
			}
			return d.request(api, method, callback, true)
		}
		return nil, code, fmt.Errorf("request failed,code: %d, message: %s", code, utils.Json.Get(res.Body(), "message").ToString())
	}
	return res.Body(), 200, nil
}
