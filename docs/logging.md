* Query service request logging

* Date: 2022-01-13

## Summary

In order to better support serverless environments the Query service needs to be able to isolate debugging to individual sessions and permit such data to be accessed by the session without allowing access to other user's data.  Request-based logging permits this.

The default request-based trace is backed by a temporary file (created in the system temporary directory) limited to \_MAX\_TRACE\_SIZE and is reported as a new "log" element in the Query service response.  "log" is an array of strings.  When \_MAX\_TRACE\_SIZE is reached for a request, the temporary file is truncated and logging continues.  When this happens the first element in "log" indicates that truncation has occured.

## How to use in the code

To enhance existing logging (via the logging package) to be available as request logging, the logging function prototypes have been moved into the logging.Log interface.  The equivalent logging package functions now accept an optional final argument.   If this final argument is present and it is cast-able to logging.Log, it is called with the other arguments in addition to the normal processing.

The execution Context provides suitable functions for request-based logging.

Pass a context pointer to the logging package function to add the ability to log to either or both locations.

To log exclusively to the query.log, don't pass the additional parameter and to log exclusively to the request-based log, call the Context function (appropriate for the level) directly.

e.g.
    logging.Debuga(func() string{ return "example possible logged to both" }, context)
    logging.Debugf("example possible logged to both: %v", value, context)

    logging.Debuga(func() string{ return "example for query.log only"})
    logging.Debugf("example for query.log only: %v", value)

    context.Debuga(func() string{ return "example for request log only"})
    context.Debugf("example for request log only: %v", value)


All existing logging is unaffected.  The general challenge for supporing request-based logging is access to a suitable context reference.  datastore.QueryContext has been enhanced to include the logging.Log interface, as have other contexts.  Mixed use functions that do want to support request logging but may be called outside of a request should try and use a dummy provider - e.g. logging.NULL\_LOG - so individual logging calls don't need checking for a valid object.  e.g.

  func eg1(log logging.Log) {    // if adding arguments, the type need only be logging.Log to avoid circular dependencies
    log.Infoa(func() string{ return "Example" })
  }

  ...
      eg1(context)
  ...
      eg1(logging.NULL_LOG)
  ...


## How to enable for a request

In the simplest form, just set the loglevel request parameter to the desired logging level.

There is support for filtering at DEBUG and TRACE levels, with the filters being matched against the "((function|file:line))" text appended to messages at these levels.  (This differs from query.log filtering which is on the full file pathname only.)  The first matching filter applies (no combining); filters are regular expressions.  A filter starting with '-' means "exclude if matching this pattern" otherwise it means "include if matching this pattern".  To specify a '-' as the first character in a filter it can be escaped, or simply start the pattern with '.' (since the strings matched against can never start with a '-'; they always start with '(').

Whilst a filters can be specified at other logging levels, they will only be applied to DEBUG and TRACE level messages.

You can additionally specify a logging provider; the full syntax of the loglevel string is:

[provider:]level[:filter]

At implementaion, the following are valid providers: builtin (which is the default), file and null.

e.g. URL snippets:

...&loglevel=info&...                     # plain INFO level using the default (i.e. "builtin") provider

...&loglevel=debug:Fetch&...              # debug including only matches for "Fetch" using the default provider

...&loglevel=builtin:-memcached;Fetch...  # debug excluding matches for "memcached" that match "Fetch" using "builtin" provider

...&loglevel=file:info&...                # plain INFO level using "file" provider
