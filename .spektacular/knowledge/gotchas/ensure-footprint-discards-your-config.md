# EnsureFootprint discards the config you pass it when a healthy repo.yaml already exists

`repo.EnsureFootprint(root, repoCfg)` (`internal/repo/footprint.go`) reads
like a scaffold call that writes the config you hand it, and on a fresh
folder it does. But it takes three paths, and only two of them use your
argument:

- no `repo.yaml` at `root` — your `repoCfg` is written out, status `created`;
- a `repo.yaml` that fails to parse — your `repoCfg` overwrites it, status
  `repaired`;
- a `repo.yaml` that parses — **your argument is thrown away** (`repoCfg =
  loaded`) and the file on disk is left exactly as it was.

The third case is the surprise, and it is deliberate: a healthy existing
config is the authority for its own footprint, so one project cannot
clobber a repo that another project already initialized. The consequence is
that the call is a no-op for your descriptive fields on the second and every
later invocation, and it fails silently — you get a status of `unchanged` or
`repaired` and a valid footprint, just not the metadata you asked for.

So never treat `EnsureFootprint` as the way to write a repo's description,
role, tags or source. Write those in a separate step afterwards: read the
config back from `<root>/repo.yaml` with `config.RepoConfigFromYAMLFile`,
apply your fields to what you read, and write it out only if something
actually changed. That is what `repo.Register` (`internal/repo/register.go`)
does, and it is why registering an already-footprinted repo still updates
its metadata.

Note also that the status is about the *footprint*, not about your config:
`repaired` is returned when the knowledge directories or their READMEs were
missing, even though `repo.yaml` was fine and your argument was ignored.
Do not read a non-`unchanged` status as evidence that your values landed.
