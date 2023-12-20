FROM registry-vpc.cn-shenzhen.aliyuncs.com/mlj/ops-public:mlj-deploy-go-noproxy
COPY app /app/
WORKDIR /app/
CMD ["app"]