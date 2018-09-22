import {combineReducers} from 'redux'
import { reducer as form } from 'redux-form'
import {ui} from './ui'
import {error} from './error'
import {session} from './session'
import {poets} from './poets'
import {issuesByVolume} from './issues'
import {poems} from './poems'
import {codeByPoetId} from './code'


export const reducers = combineReducers({
  form,
  error,
  poets,
  poems,
  issuesByVolume,
  codeByPoetId,
  session,
  ui,
})
