
Build Process:
-------------

This package is built using two stage container model. This is to eliminate platform/OS level inconsistencies and produce a smaller footprint.

A "build container" is created using Dockerfile.build where the desired GO dependecies are downloaded into. Once the GO compilation env is set up, this source package is pushed to the build container and built inside the container. Then the generated binaries are pulled from the build container to the host machine. The Dockerfile is used to build the final deployment container to include these generated binaries.

Dockerfile.build is used to create build container and compile this pacakge.

compile_with_gobuilder.sh has the instructions to compile inside the build container.

get_binaries.sh is used to extract the binaries from the build container when it is run.
The above steps are packaged into compile.sh

Dockerfile is used to create a deployment container.

In summary step - 1 is to compile in a container, step - 2 is to generate the deployment container.

  Step - 1:
  ----------
  ./compile.sh

  Step - 2:
  ---------

  docker build -t <IMAGE-TAG> .


