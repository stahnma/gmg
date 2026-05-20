const dbFactory = require('./index')

class PersistenceManager {
    constructor({ pollingClient, logger }) {
        this._pollingClient = pollingClient
        this._logger = (message) => {
            if (!logger) return
            logger(message)
        }

        this.start = this.start.bind(this)
        this.stop = this.stop.bind(this)
        this._onStatus = this._onStatus.bind(this)
    }

    async start() {
        if (this._started) throw new Error('Already started!')
        this._started = true

        this.db = dbFactory.createDb()

        this.db.exec(`
            CREATE TABLE IF NOT EXISTS temperature_log (
                temperature_log_id integer PRIMARY KEY,
                timestamp integer UNIQUE,
                grill_temperature integer(2) NOT NULL,
                food_temperature integer(2) NULL
            );
        `)

        // Prepare once, reuse on every status event (better-sqlite3 best practice).
        // INSERT OR IGNORE handles the rare case of two status updates landing
        // in the same epoch second (UNIQUE constraint on timestamp) without
        // throwing — previously the throw bubbled out as an unhandled rejection.
        this._insertStatus = this.db.prepare(`
            INSERT OR IGNORE INTO temperature_log (timestamp, grill_temperature, food_temperature)
            VALUES (strftime('%s','now'), @grill_temperature, @food_temperature)
        `)

        this._logger('Starting Persistence Manager...')
        this._pollingClient.on('status', this._onStatus)
    }

    async stop() {
        if (!this._started) throw new Error('Already stopped!')
        this._started = false
        this._pollingClient.removeListener('status', this._onStatus)
        this.db.close()
    }

    async _onStatus(status) {
        if (!status.isOn) {
            return
        }

        this._insertStatus.run({
            grill_temperature: status.currentGrillTemp,
            food_temperature: status.currentFoodTemp,
        })
    }
}

module.exports = PersistenceManager
