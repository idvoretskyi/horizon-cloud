require 'rethinkdb'
include RethinkDB::Shortcuts

$c = r.connect.repl

template = {
  hzc_api: {
    projects: [{name: 'Users', multi: true}],
    domains: [{name: 'Project', multi: false}],
    users: [{name: 'PublicSSHKeys', multi: true}]
  }
}

PP.pp r(template.keys).set_difference(r.db_list).foreach{|x| r.db_create(x)}.run
template.each {|db_name, tables|
  db = r.db(db_name)
  PP.pp r(tables.keys).set_difference(db.table_list).foreach {|x|
    db.table_create(x)
  }.run
  tables.each {|table_name, indexes|
    table = db.table(table_name)
    PP.pp table.index_list.do {|lst|
      r(indexes).filter{|index| lst.contains(index['name']).not}.foreach {|idx|
        table.index_create(idx['name'], multi: idx['multi'])
      }
    }.run
  }
}
