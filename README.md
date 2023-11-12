# MySQL DBA Operator

This operator does **not** provision MySQL database servers.
It works against and within existing database servers.
It helps user applications by allowing provisioning of individual
databases and database users inside an existing MySQL database server.

## Types

### AdminConnection

Database servers can be provisioned separately and then made available
for use with this operator by defining an <code>AdminConnection</code>.
<code>AdminConnection</code> resources can be created in any namespace 
separate from user applications.

Sample:

<pre>
apiVersion: mysql.apps.cuppett.dev/v1alpha1
kind: AdminConnection
metadata:
  name: db1
  namespace: cuppett
spec:
  host: 172.25.234.155.nip.io
  adminPassword:
    secretKeyRef:
      name: mysql
      key: database-root-password
  allowedNamespaces:
    - tenant1
    - openshift-*
    - blog-*
</pre>

<code>adminUser</code> can be defined similarly to <code>adminPassword</code>.
The default username is 'root' and the default password is an empty string.
<code>host</code> is required to be a valid hostname.

<code>allowedNamespaces</code> is there to enable usage of the admin connection for provisioning only where desired.
By default, only the namespace containing the <code>AdminConnection</code> is permitted (and does not need specified).
Allows specifying prefix by adding a trailing '*' character (e.g. blog-*).

With each <code>AdminConnection</code> an administrative database is created and updated to track the objects
provisioned with this operator.
This database helps ensure that unique UID, name and namespace databases are created and that those previously
existing or provisioned in other namespaces are not overridden, commandeered or inadvertently removed.

The following tables are created and updated as objects are created/destroyed:

<pre>
mysql> describe zz_dba_operator.managed_databases;
+---------------+-------------+------+-----+---------+-------+
| Field         | Type        | Null | Key | Default | Extra |
+---------------+-------------+------+-----+---------+-------+
| uuid          | varchar(36) | NO   | PRI | NULL    |       |
| namespace     | varchar(64) | YES  |     | NULL    |       |
| name          | varchar(64) | YES  |     | NULL    |       |
| database_name | varchar(64) | YES  |     | NULL    |       |
| created_at    | datetime(3) | YES  |     | NULL    |       |
| updated_at    | datetime(3) | YES  |     | NULL    |       |
+---------------+-------------+------+-----+---------+-------+
6 rows in set (0.00 sec)

mysql> describe zz_dba_operator.managed_users;
+------------+-------------+------+-----+---------+-------+
| Field      | Type        | Null | Key | Default | Extra |
+------------+-------------+------+-----+---------+-------+
| uuid       | varchar(36) | NO   | PRI | NULL    |       |
| namespace  | varchar(64) | YES  |     | NULL    |       |
| name       | varchar(64) | YES  |     | NULL    |       |
| username   | varchar(32) | YES  |     | NULL    |       |
| created_at | datetime(3) | YES  |     | NULL    |       |
| updated_at | datetime(3) | YES  |     | NULL    |       |
+------------+-------------+------+-----+---------+-------+
6 rows in set (0.00 sec)
</pre>

### Database

Once you have an <code>AdminConnection</code> resource, you can create a <code>Database</code>
resource. Creating a new <code>Database</code> resource effectively triggers a <sql>CREATE DATABASE</sql> 
against the server to match the specification in your custom resource.

Sample:

<pre>
apiVersion: mysql.apps.cuppett.dev/v1alpha1
kind: Database
metadata:
  name: mydb
  namespace: customer-ns
spec:
  adminConnection:
    namespace: cuppett /* Optional */
    name: db1
  name: mydb
  characterSet: utf8
  collate: utf8_general_ci
</pre>

Modifications to either <code>characterSet</code> or <code>collate</code> trigger
changes to the database defaults. 
Updates to <code>name</code> are rejected by a
validating webhook. 

### DatabaseUser

Finally, you can create a <code>DatabaseUser</code> resource to programmatically create
users and control a few attributes and permissions for the user. 
This object used to drive <sql>CREATE USER</sql> and <sql>ALTER USER</sql> operations within
the database.

Sample:
<pre>
apiVersion: mysql.apps.cuppett.dev/v1alpha1
kind: DatabaseUser
metadata:
  name: cuppett
  namespace: customer-ns
spec:
  adminConnection:
    namespace: cuppett /* Optional */
    name: db1
  username: cuppett
  identification:
    authPlugin: ''
    clearText: true
    authString:
      secretKeyRef:
        name: db-password
        key: password
  databasePermissions:
  - databaseName: mydb
    grants: /* Optional */
    - SELECT
    - INSERT
    - UPDATE
    - DELETE
</pre>

<code>databasePermissions</code> is a list of <code>Database</code> object names in the cluster (not names in the database server).
This allows for maintaining correct constraints and permission controls via both systems (Kubernetes and MySQL).

> NOTE: Optional <code>authString</code> references a <code>v1.Secret</code> created by the user.
The <code>v1.Secret</code> will have <code>ownerReferences</code> updated to belong to the operator once consumed.
This is to facilitate one-use passwords and automatically clean them up or scrub them when the user is
removed/dropped.

## Development & Testing

### Prerequisites

* [Operator SDK](https://sdk.operatorframework.io/docs/installation/)
* [Kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)
* [Testcontainers for Go](https://golang.testcontainers.org/)
* [Docker](https://docs.docker.com/get-docker/) -or- [Podman](https://podman.io/getting-started/installation)

### Testing

Ensure you've installed the prerequisites above.

```bash
make test
```

### Lineage

Operator originally built using [Operator SDK 1.3.0](https://v1-3-x.sdk.operatorframework.io/)<br />
Operator currently built using [Operator SDK 1.32.0](https://v1-29-x.sdk.operatorframework.io/)
