require 'rethinkdb'
include RethinkDB::Shortcuts

r.connect.repl

template = {hzc_api: {configs: [], domains: ['Project'], users: ['PublicSSHKeys']}}

PP.pp r(template.keys).set_difference(r.db_list).foreach{|x| r.db_create(x)}.run
template.each {|db_name, tables|
  db = r.db(db_name)
  PP.pp r(tables.keys).set_difference(db.table_list).foreach {|x|
    db.table_create(x)
  }.run
  tables.each {|table_name, indexes|
    table = db.table(table_name)
    PP.pp r(indexes).set_difference(table.index_list).foreach {|x|
      table.index_create(x)
    }.run
  }
}
