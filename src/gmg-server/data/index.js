const Database = require('better-sqlite3')
const Path = require('path')

let db

module.exports.initialize = ({ logger }) => {
   const db_path = Path.join(__dirname, './grill_data.db')

   logger('Initializing db: [%s]', db_path)

   // better-sqlite3 opens synchronously and exposes a synchronous API
   // backed by libuv internally. No driver wiring, no Promise dance.
   db = new Database(db_path)
}

module.exports.createDb = () => db
