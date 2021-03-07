package video_proxy

type HostPort struct {
	Host string
	Port int16
}

type VideoReplicasProvider interface {
	GetReplicas() []HostPort
}

func main() {

}