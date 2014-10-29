Sample data for tests, demos, tutorials, etc.

Please do NOT write tests against the "tutorial" dataset.  It will change over time to better support the tutorial.

Note: if you modify existing data in these subdirectories, you may
break existing "black box" test cases.  An alternative would be to
make your own copied subdirectories.

Black box test cases are stored as just more JSON data in the cases
subdirectory.  This allows us to use tuqtng to inspect the test cases
of tuqtng (e.g., count the number of test cases that use "GROUP BY").
