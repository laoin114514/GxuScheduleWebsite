# ============================================================================
# 三阶段构建：前端(Vue/Vite) -> 后端(Go) -> 运行镜像(Go 同时托管官网 + 接口)
#
# 最终镜像里：
#   /app/schedule-server   Go 二进制
#   /app/web/              官网静态产物（index.html + assets）
# 启动后 http://<host>:8080/        = 官网
#        http://<host>:8080/api/v1/ = 更新服务接口
# 同源部署，官网不需要配 VITE_API_BASE_URL，也不存在跨域问题。
# ============================================================================

# ---------- 阶段 1：构建前端 ----------
FROM m.daocloud.io/docker.io/library/node:22-alpine AS web-builder

WORKDIR /web

# 依赖单独分层：只改业务代码时不会重装依赖
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY frontend/ ./

# 同源托管，留空即可；要单独部署官网时再覆盖成后端域名
ARG VITE_APP_KEY=schedule
ARG VITE_API_BASE_URL=
ENV VITE_APP_KEY=${VITE_APP_KEY} \
    VITE_API_BASE_URL=${VITE_API_BASE_URL}

# 需要覆盖真实截图时，把 VITE_SHOT_* 写进 frontend/.env.production（会被一起打进构建）
RUN npm run build

# ---------- 阶段 2：构建后端 ----------
FROM m.daocloud.io/docker.io/library/golang:1.22-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.org \
    CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/schedule-server .

# ---------- 阶段 3：运行 ----------
FROM m.daocloud.io/docker.io/library/alpine:3.19

RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.aliyun.com#g' /etc/apk/repositories \
    && apk add --no-cache tzdata ca-certificates

ENV TZ=Asia/Shanghai \
    WEB_DIR=/app/web
WORKDIR /app

COPY --from=builder /out/schedule-server .
COPY --from=web-builder /web/dist ./web

EXPOSE 8080
CMD ["/app/schedule-server"]
