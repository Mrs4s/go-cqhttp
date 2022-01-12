#!/bin/sh
if [ -f /data/install.lock ];then
    touch /data/install.lock
else
     cp -f /usr/bin/cqhttp /data/cqhttp
fi
if [ "$UPDATE" = "1" ];then
  cp -f /usr/bin/cqhttp /data/cqhttp
fi
touch /data/install.lock
chmod +x /data/cqhttp
/data/cqhttp