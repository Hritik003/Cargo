package common

type Container struct {
	Image      string    `json:"image"`
	Registry   string    `json:"registry"`
	Username   string    `json:"username"`
	Password   string    `json:"password"`
	Port       string    `json:"port"`
	Env        []Setting `json:"env"`
	PullPolicy string    `json:"pullPolicy"`
}
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
