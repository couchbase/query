# go_cbq
CLI 

List of the available command line options


List of available shell commands 


PARAMETERS :

Prefix | Parameter type
-------|------
- | Query Parameter
no prefix | Predefined (Built-in) Session Variable
$ | User Defined Session Variable
-$ | Named Parameters

List of Predefined Parameters : limit, histfile, histsize, autoconfig and query_creds

Example : 

./go_cbq -ne -c=beer-sample:pass -u=Administrator

select * from `beer-sample` limit 1;

\CONNECT localhost:9498;

select * from `beer-sample` limit 1;

\DISCONNECT;

\CONNECT http://localhost:9000;

DROP A QUERY NODE.. 

select * from `beer-sample` limit 1;

\SET -creds beer-sample:pass;

\PUSH -max-parallelism 4;

\PUSH -max-parallelism 8;

\ECHO -max-parallelism;

\POP -max-parallelism;

select * from `beer-sample` limit 1;

\ALIAS cmd select version();

\ALIAS cmd2 select min_version();

\ALIAS;

\ECHO -creds limit \\cmd  hmmmmm ;

\ECHO -creds \\cmd  hmmmmm ;

\\cmd2;\\cmd;

\HELP \VERSION \SET;

\VERSION;

\EXIT;







