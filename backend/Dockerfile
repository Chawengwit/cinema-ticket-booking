FROM golang:1.24.4-alpine AS build
WORKDIR /src

# security / build tools
RUN apk add --no-cache ca-certificates tzdata

# แยก step เพื่อ cache go mod
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# สร้าง binary แบบ lean + ย้าย path ออก (trimpath)
# ปิด CGO เพื่อให้ binary portable และลด dependency
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime ----
FROM alpine:3.20
WORKDIR /app

# security: เอา cert/tz เข้ามา (เผื่อเรียก https / ใช้เวลา)
RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S app && adduser -S app -G app

COPY --from=build /out/api /app/api

# security: รันด้วย non-root
USER app

EXPOSE 8080
CMD ["/app/api"]
