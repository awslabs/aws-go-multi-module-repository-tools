# Description

Utility to update a repository's module management (modman.toml) file. Supports
setting a module and version, or deleting an existing module.

# Usage

Add or update a module dependency. Sets a module version to be used in the
management file. If the module exists it will be updated. If it does not exist
it will be added.

```
editmoduledependency -s github.com/aws/smithy-go -v v1.8.2
```

Delete a module dependency. Removes a module from the management file, if it
exists. Exists with non-zero code if module does not exist.

```
editmoduledependency -d github.com/aws/smithy-go
```
