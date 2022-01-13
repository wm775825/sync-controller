APP=sync-controller
NODE=skv-node2
USER=k8s
REMOTEDIR=/home/k8s/exper/wm/registry/

build:
	go build -o ${APP} main.go

prop: build
	scp ./${APP} ${USER}@${NODE}:${REMOTEDIR}

clean:
	rm ./${APP}
