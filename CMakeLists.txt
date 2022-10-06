CMAKE_MINIMUM_REQUIRED (VERSION 2.8)
PROJECT (query NONE)

# This project is intended to be built as part of a Couchbase Server build,
# but for nominal support building stand-alone, the following variables
# need to be specified:
#
# CMAKE_MODULE_PATH - should contain a directory containing
#      FindCouchbaseGo.cmake, eg. tlm/cmake/Modules
#
# CMAKE_INSTALL_PREFIX - should point to a directory where the bin/ output
#      directory will be placed
#
# GODEPSDIR - should point to a Go workspace directory containing all
#      transitive Go dependencies
#
# In addition, projects that only require the end-user cbq utility may set
# CBQ_ONLY to enable only that target.

INCLUDE (FindCouchbaseGo)

IF (BUILD_ENTERPRISE)
    SET (_gotags "enterprise")
ELSE (BUILD_ENTERPRISE)
    SET (_gotags "community")
ENDIF(BUILD_ENTERPRISE)

SET (GOVERSION 1.18)

SET (LDFLAGS)
IF (APPLE)
  # Suppress warning from FindCouchbaseGo
  IF (${GOVERSION} VERSION_LESS ${GO_MAC_MINIMUM_VERSION})
    SET (GOVERSION ${GO_MAC_MINIMUM_VERSION})
  ENDIF ()

  # On OS X 10.11 (El Capitan) upwards we can no longer use DYLD_LIBRARY_PATH to locate
  # runtime dependencies. Instead add the location of libcurl etc to the actual
  # binary itself.
  SET (LDFLAGS " ${LDFLAGS} -extldflags '-Wl,-rpath,@executable_path/../lib'")
ENDIF ()

IF (NOT CBQ_ONLY)
  GoYacc (GOVERSION 1.18 TARGET n1ql-yacc YFILE "${PROJECT_SOURCE_DIR}/parser/n1ql/n1ql.y")

  GET_FILENAME_COMPONENT (CURL_LIBRARY_DIR "${CURL_LIBRARIES}" DIRECTORY)
  GoInstall (TARGET cbq-engine PACKAGE github.com/couchbase/query/server/cbq-engine
    GOPATH "${PROJECT_SOURCE_DIR}/../../../.." "${GODEPSDIR}"
    DEPENDS n1ql-yacc proto-generated INSTALL_PATH bin
    CGO_INCLUDE_DIRS "${CURL_INCLUDE_DIR}"
    CGO_LIBRARY_DIRS "${CURL_LIBRARY_DIR}"
    LDFLAGS "${LDFLAGS}"
    GOVERSION ${GOVERSION}
    GOTAGS ${_gotags})

  GoInstall (TARGET cbq.old PACKAGE github.com/couchbase/query/shell/cbq.old
    GOPATH "${PROJECT_SOURCE_DIR}/../../../.." "${GODEPSDIR}"
    DEPENDS n1ql-yacc cbq-engine INSTALL_PATH bin
    GOVERSION ${GOVERSION})
ENDIF (NOT CBQ_ONLY)

GoInstall (TARGET cbq PACKAGE github.com/couchbase/query/shell/cbq
  GOPATH "${PROJECT_SOURCE_DIR}/../../../.." "${GODEPSDIR}"
  INSTALL_PATH bin
  GOVERSION ${GOVERSION})
