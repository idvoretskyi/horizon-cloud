export function attachApi(app) {
  app.get('/', (req, res) => {
    res.send('foo')
  })
}

