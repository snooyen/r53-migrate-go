# Route53 Migration Utility

A CLI utility to aid the migration of R53 records from an old hosted zone to a new one.

Outputs 4 JSON files:
* `old.json` - A list of all records in the old hosted zone
* `new.json` - A list of all records in the new hosted zone
* `missing.json` - A list of missing records in the new hosted zone
* `mismatched.json` - A list of mistmached records (same name and type but different property values)

# Usage
```
Usage of ./bin/r53-migrate:
      --aws-profile-new string        AWS profile to use for new records
      --aws-profile-old string        AWS profile to use for old records
      --dump-json                     Dump json (default true)
      --hosted-zone-name-new string   Hosted zone name to use for new records (default "mydomain.com.")
      --hosted-zone-name-old string   Hosted zone name to use for old records (default "mydomain.com.")
      --skip-new                      Skip new records
```

# Build and Run

```shell
$ make build
$ ./bin/r53-migrate \
    --aws-profile-old=my-old-aws-profile
    --aws-profile-new=my-new-aws-profile
    --hosted-zone-name-old=mydomain.com.
    --hosted-zone-name-new=mydomain.com.
```
