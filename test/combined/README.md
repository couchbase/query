## Combined testing

This is a basic test rig for running full-build testing.  It isn't really a unit test but permits some engineering coverage of
continual load testing.

The testing *deliberately* limits use of server packages to just _logging_ in order to not expose the test to general Query code
changes.

The programme may be invoked specifying `-config path_to_config_file` else `config.json` is loaded from the current directory.
See comments in the default `config.json` for details on configuring the test and its instance.

Statements may be:

1. Any `.sql` file under the configured location (see `config.json`) is loaded and run as-is.
2. If specified, basic random statements for the instance may be generated.
3. Any `.tpl` file under the configured location (see `config.json`) is used as a template to generate statements. 
   The `text/template` library is used for the templates; see the example `templates/eg.tpl` file for template specific information.
