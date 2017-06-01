readonly PROGNAME=$(basename $0)
readonly PROGDIR=$(readlink -m $(dirname $0))
readonly ARGS="$@"

cmdline() {
    local arg=
    for arg
    do
        local delim=""
        case "$arg" in
            --artifact_version)     args="${args}-a ";;
            --container_repo)       args="${args}-c ";;
            --image)                args="${args}-i ";;
            --push)                 args="${args}-p ";;
            *) [[ "${arg:0:1}" == "-" ]] || delim="\""
                args="${args}${delim}${arg}${delim} ";;
        esac
    done
 eval set -- $args

while getopts "a:c:i:hp" OPTION
    do
         case $OPTION in
         a)
             ARTIFACT_VERSION=$OPTARG
             ;;
         c)
             readonly CONTAINER_REPO=$OPTARG
             ;;
         h)
             usage
             exit 0
             ;;
         i)
             readonly IMAGE_NAME=$OPTARG
             ;;
         p)
             readonly OPERATION="build_push"
             ;;
        esac
    done
}


main() {
  local rc=0
  cmdline ${ARGS}
  source ./version.properties
  echo "ARTIFACT_VERSION: "$ARTIFACT_VERSION
  echo "IMAGE_NAME="$IMAGE_NAME
  # pipe variable to file for use in downstream job
  echo "IMAGE_NAME="$IMAGE_NAME >> env.properties
  echo "CONTAINER_REPO="$CONTAINER_REPO
  # pipe variable to file for use in downstream job
  echo "CONTAINER_REPO="$CONTAINER_REPO >> env.properties

  # collect git description for appending to docker tag
  pwd
  COMMIT_COUNT=$(git rev-list $ARTIFACT_VERSION.. --count)
  GIT_COMMIT_HEAD=$(git rev-parse HEAD | cut -c1-8)
  echo "COMMIT_COUNT="$COMMIT_COUNT
  echo "GIT_COMMIT_HEAD="$GIT_COMMIT_HEAD

  export IMAGE_TAG="$ARTIFACT_VERSION.$COMMIT_COUNT.$GIT_COMMIT_HEAD"
  echo "IMAGE_TAG="$IMAGE_TAG
  # pipe variable to file for use in downstream job
  echo "IMAGE_TAG="$IMAGE_TAG >> env.properties

  build_container

  if [ $OPERATION == "build_push" ]
    then
    push_container
    rc=$(($rc+$?))
  fi

  rc=$(($rc+$?))
  result=$?
  return $result
}

build_container() {
  echo "*** Begin container build"
  echo "docker build -t $CONTAINER_REPO/$IMAGE_NAME:$IMAGE_TAG ./"
  docker build -t $CONTAINER_REPO/$IMAGE_NAME:$IMAGE_TAG ./
  if [ $? -ne 0 ]
  then
    echo "error building $CONTAINER_REPO/$IMAGE_NAME:$IMAGE_TAG"
    exit -1
  fi
  result=$?
  return $result
}

push_container() {
  echo "*** Begin container push"
  echo "docker push $CONTAINER_REPO/$IMAGE_NAME:$IMAGE_TAG"
  docker push $CONTAINER_REPO/$IMAGE_NAME:$IMAGE_TAG
  if [ $? -ne 0 ]
  then
    echo "error pushing $CONTAINER_REPO/$IMAGE_NAME:$IMAGE_TAG"
    exit -1
  fi
  result=$?
  return $result
}

usage() {
    cat <<- EOF

  usage: $PROGNAME [-h]
  usage: $PROGNAME BUILD_SWITCH OPTIONS

  This script will build a container based on ./dockerfile and grab the git description and tag it based on the artifactversion and a substring of the git descriptio.

  -h --help                        show this help

  BUILD_SWITCH:
     -a --artifact_version                 The repo to store the container
     -c --container_repo                   The repo to store the container
     -i --image_name                       The container image name

  OPTIONS:
     -p --push_image                     If -p is passed the image will be pushed


  Examples:
     $PROGNAME -a "1.0.0" -c "dockerhub.cisco.com/vnf-docker-dev" -i "tric-sample-service"
     $PROGNAME -a "1.0.0" -c "dockerhub.cisco.com/vnf-docker-dev" -i "tric-sample-service" -p
EOF
}

main
