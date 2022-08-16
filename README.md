# version

<!-- https://github.com/open-beagle/awecloud-btel-sdk -->

```bash
git remote add upstream git@github.com:open-beagle/awecloud-btel-sdk.git

git fetch upstream

git merge master
```

## debug

```bash
bash .beagle/dist.sh
rm -rf .tmp/awecloud-btel-sdk.out
```

## realse

```bash
# 新建一个Tag
git tag v1.0.0-beagle.0

# 推送一个Tag ，-f 强制更新
git push -f origin v1.0.0-beagle.0

# 删除本地Tag
git tag -d v1.0.0-beagle.0
```

