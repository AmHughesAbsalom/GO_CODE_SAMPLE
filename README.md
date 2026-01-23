<h1> Tournament Bracket</h1>

![alt](https://github.com/AmHughesAbsalom/GO_CODE_SAMPLE/blob/main/assets/ROAD_TO_PLAYOFSS_4-03.jpg?raw=true)
This code implements a sports playoffs/tournament bracket management system with database operations. The working mechanism is as follows:

<h2> Core Functionality</h2> 
Creates tournament brackets - The CreatePlayoffs method generates elimination-style playoff brackets based on:

<ul style="line-height: 2.5;">
  <li>Number of conferences (1, 2, 4, or 8)</li>
  <li> Teams per conference (the limit parameter)</li>
  <li>Season identifier</li>
</ul>

<h3>Tournament Structure:</h3>
<ul style="line-height: 2.5;">
  <li>Uses best-of-3 game series for each matchup</li>
  <li>Teams are seeded by their standings (points/ranking)</li>
  <li>Higher seeds face lower seeds (1st vs last, 2nd vs 2nd-to-last, etc.)</li>
  <li>Winners advance through rounds until reaching the finals</li>
</ul>

<h3>Key Operations</h3>
<b>CreatePlayoffs:</b> Generates the entire bracket structure by:
<ul style="line-height: 2.5;">
  <li>Validating the season doesn't already have playoffs</li>
  <li>Fetching top teams from standings table</li>
  <li>Pairing teams strategically (reversing away teams to match high vs low seeds)</li>
  <li>Creating database records for all games across all rounds</li>
  <li>Initially only populating first-round matchups with team details; later rounds start with placeholder UUIDs</li>
</ul>

<b>ListPlayoffs:</b> Retrieves playoff data organized as a 3D structure: [rounds][fixtures][games]

<b>UpdatePlayoffs:</b> Records game winners and automatically:

<ul style="line-height: 2.5;">
  <li>Marks the winner in the database</li>
  <li>Advances winning teams to the next round</li>
  <li>Updates subsequent matchups when both teams in a pairing have won</li>
</ul>

<b>UpdatePlayoffsToNull:</b> Removes the specified team that may have been either intentionally or accidentally updated to the winners(next round) section hence reverting it back to null.

<b>DeletePlayoffs:</b> Removes all playoff records for a season

<h3>Technical Details</h3>
<ul style="line-height: 2.5;">
  <li>Uses PostgreSQL with transactions for data consistency</li>
  <li>Employs the sqlx library for database operations</li>
  <li>Generates UUIDs for unique identifiers</li>
  <li>Handles bracket progression logic automatically as games complete</li>
  <li>Includes extensive error handling and validation</li>
</ul>

The system essentially automates the complex logic of managing tournament brackets, from initial seeding through final championship matchup.
