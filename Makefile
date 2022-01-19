APP=sync-controller
USER=k8s
REMOTEDIR=/home/k8s/exper/wm/registry/
NODE2=skv-node2
NODE3=skv-node3
NODE4=skv-node4
NODE5=skv-node5

build:
	go build -o ${APP} main.go

prop: build
	scp ./${APP} ${USER}@${NODE2}:${REMOTEDIR}
	scp ./${APP} ${USER}@${NODE3}:${REMOTEDIR}
	scp ./${APP} ${USER}@${NODE4}:${REMOTEDIR}
	scp ./${APP} ${USER}@${NODE5}:${REMOTEDIR}

run: build
	# if not need sync
	# ./{APP} --sync=false > /tmp/sync.log 2>&1 &
	./{APP} > /tmp/sync.log 2>&1 &

clean:
	rm ./${APP}
