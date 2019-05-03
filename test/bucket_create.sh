#!/bin/bash

echo Creating Buckets

Site=http://127.0.0.1:8091/pools/default/buckets
Auth=Administrator:password
bucket=(customer orders product purchase review shellTest)
q=${1:-250}

for i in "${bucket[@]}"
do
echo curl -X POST -u $Auth -d name=$i -d ramQuotaMB=$q -d authType=sasl -d saslPassword=donotuse -d bucketType=couchbase $Site
curl -X POST -u $Auth -d name=$i -d ramQuotaMB=$q -d authType=sasl -d saslPassword=donotuse -d bucketType=couchbase $Site
done

cd filestore 

echo mkdir -p data/dimestore/product
mkdir -p data/dimestore/product

echo mkdir data/dimestore/customer
mkdir data/dimestore/customer

echo mkdir data/dimestore/orders
mkdir data/dimestore/orders

echo mkdir data/dimestore/review
mkdir data/dimestore/review

echo mkdir data/dimestore/purchase
mkdir data/dimestore/purchase

cd ../

echo Creating Users

UsersSite=http://localhost:8091/settings/rbac/users/local/
for i in "${bucket[@]}"
do

Id=${i}owner
Name=OwnerOf${i}
Password=${i}pass

echo curl -X PUT $UsersSite$Id -d name=$Name -d roles=bucket_full_access[${i}] -d password=$Password -u $Auth
curl -X PUT $UsersSite$Id -d name=$Name -d roles=bucket_full_access[${i}] -d password=$Password -u $Auth
done
