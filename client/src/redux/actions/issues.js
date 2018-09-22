import fetch from 'cross-fetch'
import {checkResponse} from './error'


export const READ_ISSUE_REQUESTED = 'READ_ISSUE_REQUESTED'
export const requestReadIssue = ({volume = 'latest'}) => dispatch => {
  const payload = {volume}
  
  dispatch({
    payload,
    type: READ_ISSUE_REQUESTED,
  })

  // make request
  fetch(
    `/api/v1/issue/${volume}`,
    {
      method: 'GET',
    })
    .then(checkResponse)
    .then(
      issue => dispatch(readIssueSuccessful(issue)),
      error => dispatch(readIssueFailed(error)),
    )
}


export const READ_ISSUE_SUCCESSFULL = 'READ_ISSUE_SUCCESSFUL'
export const readIssueSuccessful = issue => dispatch =>
  dispatch({
    payload: issue,
    type: READ_ISSUE_SUCCESSFULL,
  })

export const READ_ISSUE_FAILED = 'CREATE_ISSUE_FAILED'
export const readIssueFailed = error => dispatch => {
  error.message = 'read issue failed: ' + error.message
  
  dispatch({
    error,
    type: READ_ISSUE_FAILED,
  })  
}
