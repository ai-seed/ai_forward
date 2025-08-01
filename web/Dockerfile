# 多阶段构建：构建阶段
FROM node:20-alpine as builder

WORKDIR /app

# 复制 package.json 和锁文件
COPY package.json yarn.lock ./

# 安装依赖
RUN yarn install --frozen-lockfile

# 复制源代码
COPY . .

# 构建生产版本
RUN yarn build

# 生产阶段
FROM nginx:alpine

# 复制构建结果到 nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY nginx.conf /etc/nginx/nginx.conf

# 暴露端口
EXPOSE 80

# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]