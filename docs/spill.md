# Value Spilling

*D.Haggart, 2023-03-13*

In order to better support Data Greater than Memory and multi-tenant systems, we aim to spill ORDER BY and GROUP BY to disk.  To do this, the Values involved are written to temporary files in binary format.

Binary format is used to minimise the overhead of spilling; no conversion to/from (e.g.) JSON.

The Value interface defines ReadSpill & WriteSpill functions that read from or write to the temporary file (via the io.Reader & io.Writer interfaces).  This approach allows us to write with the minimum of intermediate memory allocations and allows us to retain & spill/restore unexported fields.  (Use of packages such as the gob encoder - anything reliant on reflect - would require use of exported fields only.)

Utility functions that handle basic types (including maps & arrays) are common to the value package; individual Value types need primarily handle their structure.

The spill file content follows the form of type identification byte followed by type data including nested types following the same form.

Generally when spilling we'd expect any quota reservation to be released prior to writing to disk and a new reservation be made when being restored from disk; the containers that handle spilling deal with this.

AnnotatedMap & AnnotatedArray types enhance the basic types to handle spilling Values.  They are primarily designed to handle ORDER BY and GROUP BY use cases and rely on the Foreach method to correctly iterate over the final content.  Arrays can be sorted; multiple spill files are merged with a basic merge sort whilst content for individual files is sorted prior to spilling.

