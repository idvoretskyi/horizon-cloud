export function attachApi(app) {
  app.get('/foo', (req, res) => {
    res.send('foo')
  })
}

