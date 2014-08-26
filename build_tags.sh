find . -name "*.go" > cscope.files
find . -name "*.go" | xargs gotags > tags 
