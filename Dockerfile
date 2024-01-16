FROM registry-vpc.cn-shenzhen.aliyuncs.com/maliujia/go:go-1.19.5-alpine3.13-1
ENV GO111MODULE=on
COPY .  /root
WORKDIR  /root
RUN go build -tags static -v -x -gcflags "-N -l" -ldflags "-X main.GitTag=`date +%FT`.release -X main.BuildTime=`date +%FT%T%z`" -o bin 
CMD ["/root/bin"]

# FROM registry-vpc.cn-shenzhen.aliyuncs.com/mlj/ops-public:mlj-deploy-go-noproxy
# COPY --from=0 /root/bin /app/
# WORKDIR  /app
# CMD ["/bin"]