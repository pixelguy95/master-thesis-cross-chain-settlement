# Week2

I started experimenting with the Script language inherent in bitcoin. Then when I felt more comfortable with it I began constructing my own custom transactions.

At the end of the day I finally succeded with my first ever custom transaction. Instead of the usual P2PKH my transaction required the user to fgive the answer to a mathematical equation to claim the money

The txid can be found here:

**Send**
`05345b91bc58a8274986ecaa18c5c73c5355fa4d91858e27e831af97593c6b7e`

**Claim**
`27b2a5950bd67a96fcdb4c9dcc35e5af4bdcf215fed325dba86f804101ab5646`

After some consideration i have diceded to implement the entire project in go. Not because I have any previous experience with it but rather the fact that most other repositories like btcd and ltd are implemented using golang

What started as code clean up ended up being a project moving all functionality into the code, previously I had used the bitcoin cli to do somethings. But it would be a lot nicer doing it all from the go code.

Started constructing the contract transaction on the bitcoin side. Started of with the multi-sig type but found it to be a bit comples. Instead I wnet with the same tpye of swap as detailed in the decread repository.
