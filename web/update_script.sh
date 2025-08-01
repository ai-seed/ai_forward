#!/bin/bash

# 应用更新脚本
# 功能：拉取最新代码 -> 构建Docker镜像 -> 更新容器

BRANCH="main"
GIT_REMOTE="origin"

# 打印变量
echo "分支: $BRANCH"
echo "Git远程仓库: $GIT_REMOTE"


echo "===> 步骤1/3: 从Git拉取最新代码..."
git fetch "$GIT_REMOTE"
git checkout "$BRANCH"
git pull "$GIT_REMOTE" "$BRANCH"

# 检查git pull是否成功
if [ $? -ne 0 ]; then
    echo "错误：Git拉取失败"
    exit 1
fi

echo "===> 步骤2/3: 构建Docker镜像并启动容器..."

docker-compose up -d --build

echo "✅ 镜像构建完成"

# 检查构建是否成功
if [ $? -ne 0 ]; then
    echo "错误：Docker构建失败"
    exit 1
fi

echo "===> 步骤3/3: 检查容器是否启动成功..."
if [ $? -eq 0 ]; then
    echo "更新成功完成!"
    echo "当前运行的容器状态:"
    docker-compose ps
else
    echo "错误：容器启动失败"
    exit 1
fi