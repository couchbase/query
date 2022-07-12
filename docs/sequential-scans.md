# Sequential Scans

*D.Haggart, 2022-08-01*


Query sequential scans are built on the KV "range scans" feature and are only available when the bucket capabilities list "rangeScan" and the feature control flag 0x4000 is not set.  The mechanism used is that of an indexer that slots in along-side GSI & FTS, but at a lower priority so that actual indices are always preferred. The benefit of this approach is little/no changes were needed to the planner & optimiser in order to make use of sequential scans.


The indexer exposes an "index" called "#sequentialscan" which coordinates and runs the scan.  Each IndexScan3 object in a plan utilising this “index” will result in a coordinator go-routine being spawned which schedules the necessary KV range scans and collates results.  As KV range scans are per-v-bucket, it is necessary - except for the case where a single key is detected - to generate range scans for all of them.  (If the scan would be for a single key value then a single key-validation operation - instead of a scan - for one v-bucket is all that is generated.)  These are then added to queues to be run by dedicated global worker go-routines.  This approach aims to eliminate go-routine explosion within the Query service process.


The index is visible in the system:all_indexes results (deliberately not present in system:indexes) and the metadata includes statistics related to the use of sequential scans.


There is one range scan request queue per server, servicing all the v-bucket range scan requests for v-buckets on that server.  The number of worker go-routines is determined by the number of processors available on KV nodes: the one with the fewest dictates.  For each node with a KV service, the number of processors on the node is divided by the number of services on the node and 75% of this (minimum 1) is then taken as the number of available CPUs for range scan processing.  The first scan to run calculates this value and it remains until the Query service is restarted.  The number of servers can vary dynamically: if additional KV nodes are added, new scans will request that further workers are started to suit the current number of nodes.  A single monitor go-routine will detect when workers for the highest server number have not been used in some time (60 minutes) and will try to halt them.  Reduction only occurs from the highest server number downward since v-bucket maps are not sparse.


The idea of a limited pool of workers is to ensure the KV nodes are not overloaded with concurrent requests from the Query service.  Within each pool of workers, a map of v-buckets with active scans is maintained and is used to prevent more than one scan for a v-bucket being requested at a time.  This limit is of course per Query node, so in some configurations the KV may see multiple concurrent scan requests for the same v-bucket.


Each sequential scan begins scheduling its KV range scans from a randomly selected v-bucket starting point in an effort to distribute load and be able to handle more concurrent requests in a timely manner.


When a scan is cancelled, depending on where in the processing it is a cancellation of KV range scans may be needed.  In turn, depending on where in the processing of an individual scan the worker is when the cancellation is received, the cancellation instruction may be sent directly on the existing connection or queued for another per-server dedicated worker to cancel and the connection discarded.  It is necessary to discard the connection in some circumstances since the protocol allows for multiple response packets to a single request and there is no way to know how many nor to stop/interrupt them on the same connection.  The advantage of having a dedicated worker issue the cancellation instructions to the KV engine is that it can cache its connection thereby reducing the load on the pool.  It also handles marking the v-bucket ready for further scans and thus leaves the scan worker free to commence work on another scan whilst the cancellation is effected.


The KV range scan encapsulating object is effectively handed between coordinator and worker via two queues.  The only possibly concurrent action is when a sequential scan is cancelled (if say the request result limit has been hit) and a KV range scan operation is underway with a worker.  Effort has been put in to eradicate the need for full mutex locking, using atomically changed item status to marshal actions.


If necessary temporary files are used to spill individual KV range scan results to, in order to limit the memory requirements.  The threshold before spilling is currently 10 KiB of keys, meaning for a full scan of all v-buckets, the key memory is limited to 10 MiB per sequential scan.  Temporary files are deliberately never flushed so it is possible to make use of the filesystem cache to effectively keep all keys in memory.


If an ordered scan is requested, the coordinator handles merging the different v-bucket scan result streams though a merge sort.  Individual v-bucket scans don’t need to be fully materialised before sorting can begin since each v-bucket scan is itself ordered already.  If there is sufficient volume in a v-bucket, multiple scans are issued with advancing start keys in order to stream results without maintaining an open connection and range scan (and without occupying a worker go-routine for an extended period).
