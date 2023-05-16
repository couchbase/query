# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

CMAKE_MINIMUM_REQUIRED (VERSION 3.13)
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
    SET (NEWEVAL_DIR ../eventing-ee/evaluator)
    SET (_eval_no_default_path NO_DEFAULT_PATH)
ELSE (BUILD_ENTERPRISE)
    SET (_gotags "community")
ENDIF(BUILD_ENTERPRISE)
SET (_extra_deps)

SET (GOVERSION SUPPORTED_OLDER)
GoPrivateMod (MODULE cbftx PATH ../../../../../cbftx)
GoPrivateMod (MODULE hebrew PATH ../../../../../hebrew)
GoPrivateMod (MODULE eventing-ee)
GoPrivateMod (MODULE query-ee)

STRING(REGEX REPLACE "-[^-]*$" "-N1QL" QUERY_VERSION ${PRODUCT_VERSION})
SET (LDFLAGS "-X github.com/couchbase/query/util.VERSION=${QUERY_VERSION}")
IF (APPLE)
  # On OS X 10.11 (El Capitan) upwards we can no longer use DYLD_LIBRARY_PATH to locate
  # runtime dependencies. Instead add the location of libcurl etc to the actual
  # binary itself.
  SET (LDFLAGS " ${LDFLAGS} -extldflags '-Wl,-rpath,@executable_path/../lib'")
ENDIF ()
IF(WIN32)
    IF("${MSVC_VERSION}" LESS 1910)
        SET (LDFLAGS "${LDFLAGS} -linkmode internal")
    ENDIF()
ENDIF()

IF (NOT CBQ_ONLY)
  GoYacc (GOVERSION SUPPORTED_OLDER TARGET n1ql-yacc YFILE "${PROJECT_SOURCE_DIR}/parser/n1ql/n1ql.y")

  GET_FILENAME_COMPONENT (CURL_LIBRARY_DIR "${CURL_LIBRARIES}" DIRECTORY)

  SET(CGO_INCLUDE_DIRS "${CURL_INCLUDE_DIR}")
  SET(CGO_LIBRARY_DIRS "${CURL_LIBRARY_DIR}")

  IF (BUILD_ENTERPRISE)
    find_path(NEWEVAL_INCLUDE_DIR worker.h
        HINTS ${NEWEVAL_DIR}/worker/include)
    LIST(PREPEND CGO_LIBRARY_DIRS "${evaluator_BINARY_DIR}")
    LIST(PREPEND CGO_INCLUDE_DIRS "${NEWEVAL_INCLUDE_DIR}")
  ENDIF (BUILD_ENTERPRISE)

  GoModBuild (TARGET cbq-engine PACKAGE github.com/couchbase/query/server/cbq-engine
    DEPENDS n1ql-yacc proto-generated ${_extra_deps} INSTALL_PATH bin
    CGO_INCLUDE_DIRS "${CGO_INCLUDE_DIRS}"
    CGO_LIBRARY_DIRS "${CGO_LIBRARY_DIRS}"
    LDFLAGS "${LDFLAGS}"
    GOVERSION ${GOVERSION}
    GOTAGS ${_gotags})

  IF (BUILD_ENTERPRISE)
    ADD_DEPENDENCIES (cbq-engine jseval)
  ENDIF ()

ENDIF (NOT CBQ_ONLY)

GoModBuild (TARGET cbq PACKAGE github.com/couchbase/query/shell/cbq
  INSTALL_PATH bin
  GOVERSION ${GOVERSION})
