FROM node:20-alpine AS base
WORKDIR /app

# สร้าง user ที่ไม่ใช่ root
RUN addgroup -S app && adduser -S app -G app

# ติดตั้ง deps แบบ cache-friendly
COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

COPY . .

# -------- DEV --------
FROM base AS dev
ENV NODE_ENV=development

# ใช้ su-exec เพื่อ drop privilege หลัง chown
RUN apk add --no-cache su-exec

EXPOSE 5173

# EACCES:
# 1) chown /app เพื่อให้เขียนลง node_modules/.vite ได้
# 2) รัน vite ด้วย user app (ไม่ใช่ root)
CMD ["sh","-lc","chown -R app:app /app && exec su-exec app npm run dev -- --host 0.0.0.0 --port 5173"]

# -------- BUILD (for PROD) --------
FROM base AS build
ENV NODE_ENV=production
RUN npm run build

# -------- PROD (static serve) --------
FROM nginx:1.27-alpine AS prod
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
