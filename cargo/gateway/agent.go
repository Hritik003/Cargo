package gateway

import "goavega-software/cargo/cargo/common"

type Credentials struct {
	Type     string `json:"type"`
	UserName string `json:"username"`
	Server   string `json:"server"`
	Password string `json:"password"`
	Key      string `json:"key"`
}

type Web struct {
	Type   string `json:"type"`
	Params string `json:"params"`
}

type Db struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	UserName string `json:"username"`
	Password string `json:"password"`
}
type Agent struct {
	Id          int                `json:"id"`
	Name        string             `json:"name"`
	Csp         string             `json:"csp"`
	Credentials Credentials        `json:"credentials"`
	Containers  []common.Container `json:"containers"`
	Web         Web                `json:"web"`
	Db          Db                 `json:"db"`
}
