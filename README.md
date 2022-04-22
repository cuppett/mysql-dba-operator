# MySQL DBA Operator

This operator does **not** provision MySQL database servers.
It works against and within existing database servers.
It helps user applications by allowing provisioning of individual
databases and database users inside an existing MySQL database server.

## AdminConnection

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
  host: 172.25.234.155.xip.io
  adminPassword:
    secretKeyRef:
      name: mysql
      key: database-root-password
</pre>

<code>adminUser</code> can be defined similarly to <code>adminPassword</code>.
The default username is 'root' and the default password is an empty string.
<code>host</code> is required to be a valid hostname.

## Database

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
    namespace: cuppett
    name: db1
  name: mydb
  characterSet: utf8
  collate: utf8_general_ci
</pre>

Modifications to either <code>characterSet</code> or <code>collate</code> trigger
changes to the database defaults. Updates to <code>name</code> are rejected by a
validating webhook. 

## DatabaseUser

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
    namespace: brightframe
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
</pre>

*Note:* Optional <code>authString</code> references a <code>v1.Secret</code> created by the user.
The <code>v1.Secret</code> will have <code>ownerReferences</code> updated to belong to the operator once consumed.
This is to facilitate one-use passwords and automatically clean them up or scrub them when the user is
removed/dropped.

Operator originally built using [Operator SDK 1.7.2](https://v1-7-x.sdk.operatorframework.io/)<br />
Operator currently built using [Operator SDK 1.19.1](https://v1-19-x.sdk.operatorframework.io/)
