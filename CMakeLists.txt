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
    SET(_eval_no_default_path NO_DEFAULT_PATH)
ELSE (BUILD_ENTERPRISE)
    SET (_gotags "community")
ENDIF(BUILD_ENTERPRISE)
SET (_extra_deps)

SET (GOVERSION SUPPORTED_NEWER)

STRING(REGEX REPLACE "-[^-]*$" "-N1QL" QUERY_VERSION ${PRODUCT_VERSION})
SET (LDFLAGS "-X github.com/couchbase/query/util.VERSION=${QUERY_VERSION}")
IF(WIN32)
    IF("${MSVC_VERSION}" LESS 1910)
        SET (LDFLAGS "${LDFLAGS} -linkmode internal")
    ENDIF()
ENDIF()

SET(CGO_INCLUDE_DIRS)
SET(CGO_LIBRARY_DIRS)

IF (NOT CBQ_ONLY)
  GoYacc (GOVERSION SUPPORTED_NEWER TARGET n1ql-yacc YFILE "${PROJECT_SOURCE_DIR}/parser/n1ql/n1ql.y")

  IF (BUILD_ENTERPRISE)
    find_path(NEWEVAL_INCLUDE_DIR worker.h
        HINTS ${NEWEVAL_DIR}/worker/include)
    LIST(PREPEND CGO_LIBRARY_DIRS "${evaluator_BINARY_DIR}")
    LIST(PREPEND CGO_INCLUDE_DIRS "${NEWEVAL_INCLUDE_DIR}")
    LIST(APPEND _extra_deps jseval)

    IF (CB_BUILD_FAISS)
      LIST(APPEND _extra_deps faiss_c)
      SET(_gotags "${_gotags} vectors")
    ENDIF ()
  ENDIF (BUILD_ENTERPRISE)

  GoModBuild (TARGET cbq-engine PACKAGE github.com/couchbase/query/server/cbq-engine
    DEPENDS n1ql-yacc sigar proto-generated ${_extra_deps} INSTALL_PATH bin
    LDFLAGS "${LDFLAGS}"
    CGO_INCLUDE_DIRS "${CGO_INCLUDE_DIRS}"
    CGO_LIBRARY_DIRS "${CGO_LIBRARY_DIRS}"
    GOVERSION ${GOVERSION}
    GOTAGS "${_gotags}")

ENDIF (NOT CBQ_ONLY)

GoModBuild (TARGET cbq PACKAGE github.com/couchbase/query/shell/cbq
  INSTALL_PATH bin
  GOVERSION ${GOVERSION}
  CGO_INCLUDE_DIRS "${CGO_INCLUDE_DIRS}"
  CGO_LIBRARY_DIRS "${CGO_LIBRARY_DIRS}"
  LDFLAGS "-X github.com/couchbase/query/shell/cbq/command.SHELL_VERSION=${PRODUCT_VERSION}")

AddMetricsMetadata (JSON etc/metrics_metadata.json COMPONENT n1ql)
