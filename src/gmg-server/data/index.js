const SQLite = require('sqlite')
const sqlite3 = require('sqlite3')
const Path = require('path')

let dbPromise

module.exports.initialize = ({ logger }) => {
   const db_path = Path.join(__dirname, './grill_data.db')

   logger('Initializing db: [%s]', db_path)

   // sqlite v4 uses an options object + an explicit driver, unlike v3's
   // positional (filename, options) signature. Store the open Promise so
   // existing async consumers (await createDb()) keep working without
   // changing the bin/www boot order.
   dbPromise = SQLite.open({ filename: db_path, driver: sqlite3.Database })
}

module.exports.createDb = async () => dbPromise
