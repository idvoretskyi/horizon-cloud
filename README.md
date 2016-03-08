<img style="width:100%;" src="/github-banner.png">

# Horizon Cloud

Horizon Cloud is a cloud management service for deploying, managing, and
scaling Horizon applications. The goal is to create an experience that
will allow developers to build a RethinkDB/Horizon app on their laptop,
deploy it with a simple command, and scale the app up and down in a
click of a button.

__NOTE:__ "Horizon" is a codename that we'll likely change in the
future. See https://github.com/rethinkdb/horizon/issues/7.

Horizon Cloud will provide the following services:

- __Deployment__ -- users should be able to type something to the
  effect of `horizon deploy` to get their app online in one command.
- __Database autoscaling__ -- Horizon Cloud should allow users to easily
  scale RethinkDB and Horizon up and down with demand. We often get
  questions such as "how many shards do I need?" and "how many
  replicas do I need?" Horizon Cloud should obviate these questions
  either by automatically managing scalability, or suggesting
  administrative operations based on load.
- __Middleware autoscaling__ -- similarly to database autoscaling,
  Horizon Cloud will automatically allow scaling the number of Horizon
  servers up and down to accommodate demand.
- __Backup/restore__ -- this should be self-explanatory.
- __Rolling version updates__ -- Horizon Cloud will offer ability to
  upgrade RethinkDB/Horizon versions without interrupting the
  application.
- __Application versioning__ -- similarly to rolling version updates
  for the infrastructure, Horizon Cloud should manage application
  versioning and rollout, ideally without interrupting the
  application.
- __Environments__ -- users will be able to deploy apps into testing,
  staging, and production.
- __Monitoring__ -- Horizon Cloud should offer realtime and historical
  monitoring capabilities (number of queries, connections, amount of
  data, load, etc.)
- __Content delivery__ -- Horizon Cloud will automate distributing code
  and data around the world via multiple datacenters, CDN services,
  etc.
- __Multitenancy__ -- users should be able to deploy multiple apps and
  manage them from a single location.

Horizon Cloud will likely be based on the following rough architecture:

- __User-defined cloud providers__ -- users should be able to pick a
  cloud service where they want to host (e.g. AWS, Google Cloud,
  Azure, etc.) We may not do this on day one, but eventually they
  should be able to pick their cloud provider.
- We will __not__ run the infrastructure for them -- just offer
  management services. In other words, they'll give us their
  Amazon/Azure/etc. keys and our services will do the management. We
  will not be abstracting away the underlying provider. We'll charge
  for management, and users will pay their cloud provider directly for
  the hardware usage.
- __Containers__ -- we'll use Amazon's/Google's/etc. container
  services. Horizon applications will be dockerized, which will largely
  solve the problem of portability.
- __On-premise deployment__ -- for enterprise customers, we'll allow
  them to run Horizon Cloud on their servers onsite, and deploy the apps
  into their private cloud infrastructure (e.g. Kubernetes, Open
  Stack, etc.)

## FAQ

### Why would people use Horizon Cloud instead of deploying to AWS themselves or writing a Kubernetes script?

Deployment and management is a serious challenge for different classes of users.

- For individuals, figuring out how to get an app online is a pretty
  serious distraction; it's a pain, and most people just want to get
  the app out to share with their friends/colleagues.
- For small and medium companies, figuring out how to scale an app and
  do best-of-breed deployment practices is completely unobvious. It's
  dramatically easier to just pay for a service that manages all the
  challenges instead of figuring out how to solve them manually on top
  of a cloud provider.
- For enterprises, there is non-trivial amount of
  management/auditing/compliance work involved in app management, and
  enterprises typically have far more money than time.

### How does Horizon Cloud compare to Compose.io?

- Horizon Cloud will manage datbaase autoscaling __and__ Horizon app
  server autoscaling; it's not just about the database.
- Because we also manage applications, there will be lots of
  additional services (e.g. rolling app updates) that Compose.io can't
  provide.
- Similarly, since we manage applications users will be able to type
  `horizon deploy` to take care of full stack deployment -- something
  Compose.io can't do.
- We will __not__ be abstracting away the underlying cloud
  provider. We'll just charge for management, and won't need a large
  devops team to manage people's deployments.

### How does Horizon Cloud compare to Firebase/Parse?

Horizon Cloud is a management service for deploying an open-source stack
(RethinkDB + Horizon). Anybody could deploy this stack themselves,
Horizon Cloud will just make it dramatically easier. Users won't *need*
the service to build their application, unlike Firebase/Parse they can
just download Horizon/RethinkDB on their laptop.

### How does Horizon Cloud compare to Heroku?

Horizon Cloud to Horizon/RethinkDB is basically what Heroku was for
Rails/MySQL at its inception. The major difference here is that we
control most of the software stack, so we can have a much tighter
integration between Horizon/RethinkDB and Horizon Cloud, and offer people
a much more compelling development and operations experience.

### How does Horizon Cloud compare to Meteor Galaxy?

Horizon Cloud is very similar to Meteor Galaxy. What Galaxy is to Meteor,
Horizon Cloud is to RethinkDB+Horizon.

### How will Horizon Cloud be priced?

There will be four pricing tiers:

|                    | Starter  | SMB1              | SMB2               | Enterprise         |
| ------------------ |:---------| :-----------------|:-------------------|:-------------------|
| Horizon nodes      | 1        | unlimited         | unlimited          | unlimited          |
| RethinkDB nodes    | 1        | unlimited         | unlimited          | unlimited          |
| Datacenters        | 1        | 1                 | unlimited          | unlimited          |
| On-prem deployment | no       | no                | no                 | yes                |
| Price              | $5/month | $49/DB node/month | $149/DB node/month | $999/DB node/month |

