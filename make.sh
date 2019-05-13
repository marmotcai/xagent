#!/bin/bash

cmd=${1}
param1=${2}
case $cmd in 
    commit)
      git add .
      curtime=`date "+%Y-%m-%d:%H:%M:%S"`
      git commit -m "auto commit ${curtime}"
      git push
    ;;

    pull)
      git pull
    ;;

    build)
      sh docker-build.sh github.com/marmotcai/uploadagent $PWD/output ua
    ;;

    build-arm)
      sh docker-build.sh github.com/marmotcai/uploadagent $PWD/output ua arm
    ;;

    *)
      echo "use: sh make.sh commit"
      echo "use: sh make.sh pull"
      echo "use: sh make.sh build"
    ;;
esac

exit 0;
