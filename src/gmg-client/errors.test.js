const AppError = require('./AppError')
const InvalidCommand = require('./InvalidCommand')

describe('AppError', () => {
  test('sets message and default status 500', () => {
    const err = new AppError('something broke')
    expect(err.message).toBe('something broke')
    expect(err.status).toBe(500)
    expect(err.name).toBe('AppError')
    expect(err).toBeInstanceOf(Error)
  })

  test('accepts custom status', () => {
    const err = new AppError('not found', 404)
    expect(err.status).toBe(404)
  })

  test('has stack trace', () => {
    const err = new AppError('test')
    expect(err.stack).toBeDefined()
    expect(err.stack).not.toContain('AppError.constructor')
  })
})

describe('InvalidCommand', () => {
  test('defaults to status 400 and default message', () => {
    const err = new InvalidCommand()
    expect(err.message).toBe('Invalid Command')
    expect(err.status).toBe(400)
    expect(err.name).toBe('InvalidCommand')
  })

  test('accepts custom message', () => {
    const err = new InvalidCommand('grill is off')
    expect(err.message).toBe('grill is off')
    expect(err.status).toBe(400)
  })

  test('is instanceof AppError and Error', () => {
    const err = new InvalidCommand()
    expect(err).toBeInstanceOf(AppError)
    expect(err).toBeInstanceOf(Error)
  })
})
